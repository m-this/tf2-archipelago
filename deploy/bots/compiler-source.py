#!/usr/bin/env python3
"""Recover a pruned SourceMod compiler drop from its exact official sources."""
import argparse
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys


def run(args, cwd=None):
    return subprocess.check_output(args, cwd=cwd, text=True).strip()


def revision_for_build(repo, build):
    # SourceMod's own generate_headers.py numbers builds with rev-list --count.
    # Published branch builds are on its first-parent chain; counts increase.
    commits = run(['git', 'rev-list', '--first-parent', 'HEAD'], repo).splitlines()
    low, high = 0, len(commits)
    while low < high:
        middle = (low + high) // 2
        count = int(run(['git', 'rev-list', '--count', commits[middle]], repo))
        if count > build:
            low = middle + 1
        elif count < build:
            high = middle
        else:
            return commits[middle]
    raise ValueError(f'official branch has no first-parent build {build}; refusing another compiler')


def checkout(url, ref, target, branch=False):
    if not (target / '.git').exists():
        args = ['git', 'clone', '--filter=blob:none', '--no-checkout']
        if branch:
            args += ['--single-branch', '--branch', ref]
        args += [url, str(target)]
        subprocess.run(args, check=True)
    subprocess.run(['git', 'fetch', 'origin', ref], cwd=target, check=True)
    subprocess.run(['git', 'checkout', '--detach', 'FETCH_HEAD'], cwd=target, check=True)


def build(opt):
    match = re.fullmatch(r'(\d+\.\d+\.\d+)-git(\d+)', opt.version)
    if not match or not re.fullmatch(r'\d+\.\d+', opt.branch):
        raise ValueError('expected a pinned SourceMod X.Y.Z-gitN drop and X.Y branch')
    product, number = match[1], int(match[2])
    work = opt.work.resolve()
    work.mkdir(parents=True, exist_ok=True)
    sm = work / 'sourcemod'
    checkout('https://github.com/alliedmodders/sourcemod.git', opt.branch + '-dev', sm, branch=True)
    commit = revision_for_build(sm, number)
    subprocess.run(['git', 'checkout', '--detach', commit], cwd=sm, check=True)
    actual = (sm / 'product.version').read_text().strip().split('-')[0]
    if actual != product:
        raise ValueError(f'resolved product {actual} does not match requested {product}')
    subprocess.run(['git', 'submodule', 'update', '--init', '--depth', '1', 'sourcepawn'], cwd=sm, check=True)
    sp = sm / 'sourcepawn'
    subprocess.run(['git', 'submodule', 'update', '--init', '--recursive', '--depth', '1'], cwd=sp, check=True)
    sp_commit = run(['git', 'rev-parse', 'HEAD'], sp)
    amb = work / 'ambuild'
    checkout('https://github.com/alliedmodders/ambuild.git', opt.ambuild, amb)
    env = {**os.environ, 'PYTHONPATH': str(amb) + os.pathsep + os.environ.get('PYTHONPATH', '')}
    # The unified SourceMod build labels spcomp with SourceMod's build version.
    # Standalone SourcePawn needs the same label supplied explicitly.
    (sp / 'product.version').write_text(f'{product}.{number}\n')
    # Old compiler trees use clang-only warning policies. Match our existing
    # standalone toolchain bootstrap: warnings stay visible, not build errors.
    script = sp / 'AMBuildScript'
    script.write_text(script.read_text().replace("'-Werror',", "'-Wno-error',"))
    configure = sp / 'configure.py'
    configure.write_text(configure.read_text().replace('from pkg_resources import parse_version', 'from packaging.version import parse as parse_version'))
    obj = sp / 'objdir'
    obj.mkdir(exist_ok=True)
    subprocess.run([sys.executable, '../configure.py', '--enable-optimize', '--targets=x86_64'], cwd=obj, env=env, check=True)
    subprocess.run([sys.executable, '-c', 'from ambuild2.run import cli_run; cli_run()', '--jobs', '4'], cwd=obj, env=env, check=True)
    compiler = obj / 'spcomp/linux-x86_64/spcomp'
    result = subprocess.run([str(compiler)], capture_output=True, text=True)
    if f'SourcePawn Compiler {product}.{number}' not in result.stdout + result.stderr:
        raise ValueError('source-built compiler does not report the requested version')
    scripting = opt.output.resolve() / 'addons/sourcemod/scripting'
    include = scripting / 'include'
    include.mkdir(parents=True, exist_ok=True)
    shutil.copytree(sp / 'include', include, dirs_exist_ok=True)
    shutil.copytree(sm / 'plugins/include', include, dirs_exist_ok=True)
    generated = work / 'version-headers'
    generated.mkdir(exist_ok=True)
    subprocess.run([sys.executable, str(sm / 'tools/buildbot/generate_headers.py'), str(sm), str(generated)], check=True)
    shutil.copy2(generated / 'version_auto.inc', include / 'version_auto.inc')
    shutil.copy2(compiler, scripting / 'spcomp64')
    provenance = {'requested_drop': opt.version, 'sourcemod_commit': commit,
                  'sourcepawn_commit': sp_commit, 'ambuild_commit': opt.ambuild,
                  'build_number': number, 'recovery': 'official source; rev-list build identity verified'}
    (opt.output / 'compiler-source.json').write_text(json.dumps(provenance, indent=2) + '\n')
    print(json.dumps(provenance), flush=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--version', required=True)
    parser.add_argument('--branch', required=True)
    parser.add_argument('--ambuild', required=True)
    parser.add_argument('--work', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    build(parser.parse_args())
