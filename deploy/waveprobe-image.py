#!/usr/bin/env python3
"""Prepare a local game image whose layers can be shared by isolated workers."""
import argparse
import datetime as dt
import importlib.util
import json
import hashlib
import os
import re
from pathlib import Path
import secrets
import shutil
import subprocess
import time
import uuid

REPO = Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location('stability_run', REPO / 'deploy/stability-run.py')
runner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(runner)


def community_spec():
    files=[REPO/'gamedata/community.json',REPO/'launcher/internal/installer/community_hash.go']
    return hashlib.sha256(b''.join(path.read_bytes() for path in files)).hexdigest()


def add_community(tag, archives, binary, output):
    """Append validated packs once; no per-worker extraction or writable sharing."""
    name='tf2ap-community-'+uuid.uuid4().hex[:12]
    config=json.loads(subprocess.check_output(['docker','image','inspect','--format','{{json .Config}}',tag],text=True))
    output.mkdir(parents=True,exist_ok=True)
    subprocess.run(['docker','run','-d','--name',name,'-v',str(archives.resolve())+':/tf2ap-packs:ro',
                    '--entrypoint','sleep',tag,'infinity'],check=True,stdout=subprocess.DEVNULL)
    try:
        subprocess.run(['docker','cp',str(binary),name+':/tmp/tf2ap-communityfiles'],check=True)
        with (output/'community-install.log').open('w') as logs:
            subprocess.run(['docker','exec',name,'/tmp/tf2ap-communityfiles','-archives','/tf2ap-packs',
                            '-tf','/home/steam/tf-dedicated/tf'],stdout=logs,stderr=logs,check=True,timeout=1200)
        subprocess.run(['docker','exec',name,'rm','-f','/tmp/tf2ap-communityfiles'],check=True)
        subprocess.run(['docker','stop',name],check=True,stdout=subprocess.DEVNULL,timeout=60)
        subprocess.run(['docker','commit','--change','ENTRYPOINT '+json.dumps(config.get('Entrypoint') or ['/usr/bin/env']),
                        '--change','CMD '+json.dumps(config.get('Cmd') or []),
                        '--change','LABEL tf2ap.community-spec='+community_spec(),
                        '--change','LABEL tf2ap.community-dir='+json.dumps(str(archives.resolve())),name,tag],
                        check=True,stdout=subprocess.DEVNULL,timeout=1200)
    finally:
        subprocess.run(['docker','rm','-f',name],capture_output=True,timeout=60,check=False)


def prepare(image, source_volume, tag, rcon_binary, output, port=27195):
    output.mkdir(parents=True, exist_ok=True)
    name = 'tf2ap-image-' + uuid.uuid4().hex[:12]
    game = '/home/steam/tf-dedicated'
    password = secrets.token_hex(24)
    env = {**os.environ, 'SRCDS_RCONPW': password, 'SRCDS_RCON_HOST': '127.0.0.1', 'SRCDS_PORT': str(port)}
    def call(args, **kwargs):
        return subprocess.run(args, env=env, text=True, capture_output=True, check=True, **kwargs)
    def rcon(command):
        return call([str(rcon_binary), command], timeout=15).stdout
    config = json.loads(call(['docker','image','inspect','--format','{{json .Config}}',image]).stdout)
    if config.get('Volumes'):
        raise ValueError('shared worker images require a base without inherited Docker volumes')
    source_bytes = int(call(['docker','run','--rm','--log-driver','none','-v',source_volume+':/source:ro',
                             '--entrypoint','du',image,'-sb','/source']).stdout.split()[0])
    docker_root = call(['docker','info','--format','{{.DockerRootDir}}']).stdout.strip()
    if shutil.disk_usage(docker_root).free < 2*source_bytes + 30*1024**3:
        raise RuntimeError('not enough disk for the image copy, commit and 30 GiB host reserve')
    latest = runner.live_version()
    created = False
    server = None
    try:
        call(['docker','run','-d','--name',name,'--network','bridge','-p',f'127.0.0.1:{port}:27015/tcp',
              '--entrypoint','sleep',image,'infinity'])
        created = True
        call(['docker','exec','--user','root',name,'mkdir','-p',game])
        print('Copying a read-only source into one disposable image builder.', flush=True)
        # Exclude installed mods and operator settings. Community map/population
        # files remain intact; the production entrypoint installs its pinned mods.
        copy = subprocess.Popen(['docker','run','--rm','--log-driver','none','--name',name+'-transfer','-v',source_volume+':/source:ro',
            '--entrypoint','tar',image,'-C','/source','--exclude=./tf/addons*',
            '--exclude=./tf/cfg/server.cfg','--exclude=./tf/cfg/autoexec.cfg',
            '--exclude=./tf/cfg/sourcemod*','--exclude=./tf/cfg/sigsegv*',
            '--exclude=./tf/console*.log','--exclude=./tf/logs/*','-cf','-','.'], stdout=subprocess.PIPE)
        try:
            copied = subprocess.run(['docker','cp','-a','-',name+':'+game], stdin=copy.stdout,
                                    capture_output=True, timeout=1200)
            copy.stdout.close()
            if copied.returncode or copy.wait(timeout=30):
                raise RuntimeError('copy into the image builder failed')
        finally:
            subprocess.run(['docker','rm','-f',name+'-transfer'],capture_output=True,timeout=30,check=False)
            if copy.poll() is None:
                copy.terminate()
                copy.wait(timeout=30)
        call(['docker','exec','--user','root',name,'chown','-R','steam:steam',game])
        with (output/'preparation.log').open('w') as logs:
            args=['docker','exec']
            settings={'SRCDS_RCONPW':password,'SRCDS_PORT':'27015','SRCDS_STARTMAP':'mvm_decoy',
                      'SRCDS_REACH':'lan','SRCDS_LAN':'1','SRCDS_MAXPLAYERS':'32','SRCDS_BOTS':'0',
                      'SRCDS_MODS':'sigsegv-mvm','SRCDS_MOD_LOADING':'always','TF2AP_TEST_MODE':'0'}
            # Password comes through the subprocess environment, never the image config.
            env.update(settings); env['SRCDS_RCON_HOST']='127.0.0.1'
            for key in settings: args += ['-e',key]
            args += [name,*(config.get('Entrypoint') or []),*(config.get('Cmd') or [])]
            server=subprocess.Popen(args, env=env, stdout=logs, stderr=logs)
            deadline=time.monotonic()+1200
            while time.monotonic()<deadline:
                if server.poll() is not None:
                    raise RuntimeError('production entrypoint exited during image preparation')
                try:
                    # Host connects to the loopback-published preparation port.
                    prior=env['SRCDS_PORT'];env['SRCDS_PORT']=str(port)
                    try: evidence=rcon('version')
                    finally:env['SRCDS_PORT']=prior
                    current=runner.running_version(Path('/unused'),evidence)
                    staged=call(['docker','exec',name,'test','-f',game+'/tf/addons/sourcemod/extensions/sigsegv.autoload']).returncode==0
                    if current==latest and staged:
                        (output/'engine-version.txt').write_text(evidence)
                        break
                except (OSError,subprocess.SubprocessError,ValueError):pass
                time.sleep(3)
            else:raise RuntimeError('latest TF2 and pinned mod files not ready in 20 minutes')
            # This is a prepared fixture, not the fresh-install startup measurement.
            call(['docker','stop',name],timeout=60)
            server.wait(timeout=30)
        if re.search(r"Segmentation fault|FATAL ERROR|EXCEPTION_ACCESS_VIOLATION", (output/"preparation.log").read_text(errors="replace"), re.I):
            raise RuntimeError('engine crash during preparation')
        if runner.live_version()!=latest:
            raise RuntimeError('TF2 changed during preparation; rebuild the image')
        # Remove generated secrets, logs and stale process files from the snapshot.
        call(['docker','start',name])
        call(['docker','exec','--user','root',name,'sh','-c',
              'rm -f /home/steam/tf-dedicated/tf/cfg/server.cfg /home/steam/tf-dedicated/tf/console*.log /home/steam/tf-dedicated/*.pid; rm -rf /home/steam/tf-dedicated/tf/addons/sourcemod/logs'])
        call(['docker','stop',name],timeout=60)
        changes=['docker','commit','--change','ENTRYPOINT '+json.dumps(config.get('Entrypoint') or ['/usr/bin/env']),
                 '--change','CMD '+json.dumps(config.get('Cmd') or []),
                 '--change','LABEL tf2ap.prepared-version='+latest,
                 '--change','LABEL tf2ap.base-image='+call(['docker','image','inspect','--format','{{.Id}}',image]).stdout.strip(),name,tag]
        print('Saving the prepared game as shared Docker image layers.',flush=True)
        call(changes,timeout=1200)
        meta={'image':tag,'image_id':call(['docker','image','inspect','--format','{{.Id}}',tag]).stdout.strip(),
              'server_version':latest,'prepared_utc':dt.datetime.now(dt.timezone.utc).isoformat(),
              'scope':'prepared wave fixture; fresh startup remains a separate CI check'}
        (output/'image.json').write_text(json.dumps(meta,indent=2)+'\n')
        return meta
    finally:
        if created:
            subprocess.run(['docker','rm','-f',name],capture_output=True,timeout=60,check=False)
        if server and server.poll() is None:
            server.terminate();server.wait(timeout=30)


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--image',required=True)
    parser.add_argument('--source-volume',required=True)
    parser.add_argument('--tag',required=True)
    parser.add_argument('--rcon',type=Path,required=True)
    parser.add_argument('--output',type=Path,required=True)
    opt=parser.parse_args()
    prepare(opt.image,opt.source_volume,opt.tag,opt.rcon.resolve(),opt.output)
