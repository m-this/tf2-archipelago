#!/usr/bin/env python3
"""Run a resource-bounded local sweep with shared Docker game layers."""
import argparse
import datetime as dt
import importlib.util
import json
import os
from pathlib import Path
import secrets
import shutil
import subprocess

REPO=Path(__file__).resolve().parent.parent
spec=importlib.util.spec_from_file_location('waveprobe_image',REPO/'deploy/waveprobe-image.py')
images=importlib.util.module_from_spec(spec);spec.loader.exec_module(images)


def capacity(cpu_count, memory_gib, free_gib, physical_cores=None, reserve_cores=4):
    cores=physical_cores or max(1,cpu_count//2)
    # Keep four physical cores, eight GiB RAM and thirty GiB disk for the host.
    return max(0,min(16,max(1,cores-reserve_cores),int(max(0,memory_gib-8)//3),int(max(0,free_gib-30)//2)))


def resources():
    info=json.loads(subprocess.check_output(['docker','info','--format','{{json .}}'],text=True))
    physical=set()
    for cpu in os.sched_getaffinity(0):
        root=Path(f'/sys/devices/system/cpu/cpu{cpu}/topology')
        if (root/'core_id').exists():physical.add(((root/'physical_package_id').read_text(),(root/'core_id').read_text()))
    memory=info['MemTotal']/1024**3
    for line in Path('/proc/meminfo').read_text().splitlines():
        if line.startswith('MemAvailable:'):memory=min(memory,int(line.split()[1])/1024**2)
    free=shutil.disk_usage(info['DockerRootDir']).free/1024**3
    cores=min(len(physical) or info['NCPU'],info['NCPU'])
    return {'logical_cpus':info['NCPU'],'physical_cores':cores,'available_memory_gib':memory,
            'free_disk_gib':free,'recommended_workers':capacity(info['NCPU'],memory,free,cores),
            'max_workers':capacity(info['NCPU'],memory,free,cores,reserve_cores=0)}


def main(opt):
    profile=resources()
    if opt.plan:
        print(json.dumps(profile,indent=2));return
    out=opt.output.resolve() if opt.output else REPO/'docs/audits'/dt.datetime.now(dt.timezone.utc).strftime('waveprobe-local-%Y%m%d-%H%M%S')
    out.mkdir(parents=True,exist_ok=False)
    (out/'resources.json').write_text(json.dumps(profile,indent=2)+'\n')
    workers=opt.workers or profile['recommended_workers']
    if not 1<=workers<=profile['max_workers']:
        raise ValueError(f"workers must be in [1, {profile['max_workers']}] for the available resources")
    subprocess.run(['make','embed-placeholders'],cwd=REPO,check=True)
    rcon=out/'rcon'
    subprocess.run(['go','build','-o',str(rcon),'./launcher/cmd/rcon'],cwd=REPO,check=True)
    tag=opt.prepared_image or 'tf2ap-waveprobe:prepared'
    latest=images.runner.live_version()
    cached=subprocess.run(['docker','image','inspect','--format','{{json .Config.Labels}}',tag],capture_output=True,text=True)
    labels=(json.loads(cached.stdout) or {}) if cached.returncode==0 else {}
    opt.image=opt.image or labels.get('tf2ap.base-image') or 'tf2-archipelago-srcds:latest'
    base=subprocess.check_output(['docker','image','inspect','--format','{{.Id}}',opt.image],text=True).strip()
    rebuild=opt.rebuild or not labels or labels.get('tf2ap.prepared-version')!=latest or labels.get('tf2ap.base-image')!=base
    need_packs=rebuild or labels.get('tf2ap.community-spec')!=images.community_spec()
    if (rebuild or need_packs) and profile['free_disk_gib']<60:
        raise RuntimeError('at least 60 GiB free Docker storage is needed to prepare the shared image')
    packs=opt.community_dir or Path(labels.get('tf2ap.community-dir',str(REPO/'community-content')))
    if need_packs:
        tool=out/'communityfiles'
        subprocess.run(['go','build','-o',str(tool),'./launcher/cmd/communityfiles'],cwd=REPO,
                       env={**os.environ,'CGO_ENABLED':'0','GOOS':'linux'},check=True)
        subprocess.run([str(tool),'-archives',str(packs.resolve()),'-check'],check=True)
    if rebuild:
        images.prepare(opt.image,opt.source_volume,tag,rcon,out/'preparation')
    else:
        (out/'preparation-cache.json').write_text(json.dumps({'image':tag,'server_version':latest,'cached':True},indent=2)+'\n')
    if need_packs:
        print('Installing both validated community packs into one shared layer.',flush=True)
        images.add_community(tag,packs,tool,out/'preparation')
    current=resources()
    if workers>current['max_workers']:
        raise RuntimeError('resources changed during preparation; choose fewer workers')
    env={**os.environ,'WAVEPROBE_RCONPW':secrets.token_hex(24),'WAVEPROBE_SHARED_IMAGE':'1',
         'WAVEPROBE_SRCDS_IMAGE':tag,'WAVEPROBE_SHARDS':str(workers),'WAVEPROBE_SPEED':'50',
         'WAVEPROBE_RUN_DIR':str(out),'WAVEPROBE_CPUS':'2','WAVEPROBE_MEMORY':'3g'}
    print(f'Starting {workers} isolated workers at 50x; game files share image layers.',flush=True)
    status=subprocess.call(['bash','deploy/run-waveprobe.sh'],cwd=REPO,env=env)
    verification={'expected_server_version':latest,'fresh':False}
    try:
        verification['latest_after']=images.runner.live_version()
        verification['fresh']=verification['latest_after']==latest
    except (OSError,ValueError,KeyError,TypeError) as error:
        verification['error']=str(error)
    (out/'game-build.json').write_text(json.dumps(verification,indent=2)+'\n')
    if not verification['fresh']:
        report=out/'REPORT.md'
        prior=report.read_text() if report.exists() else ''
        report.write_text('**Current TF2 build unverified or changed during this run. The wave results below are not a current-build stability result.**\n\n'+prior)
        print('TF2 build changed or could not be verified after the sweep; rerun against the current build.',flush=True)
        status=1
    raise SystemExit(status)


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--image',default=os.environ.get('WAVEPROBE_SRCDS_IMAGE'))
    parser.add_argument('--source-volume',default=os.environ.get('WAVEPROBE_SOURCE_VOLUME','tf2-archipelago-waveprobe_tf2game_waveprobe'))
    parser.add_argument('--prepared-image')
    parser.add_argument('--community-dir',type=Path,default=Path(os.environ['WAVEPROBE_COMMUNITY_DIR']) if os.environ.get('WAVEPROBE_COMMUNITY_DIR') else None)
    parser.add_argument('--workers',type=int,default=int(os.environ['WAVEPROBE_SHARDS']) if os.environ.get('WAVEPROBE_SHARDS') else None)
    parser.add_argument('--output',type=Path)
    parser.add_argument('--rebuild',action='store_true')
    parser.add_argument('--plan',action='store_true')
    main(parser.parse_args())
