#!/usr/bin/env python3
"""Record verified roadmap progress, without silently counting planned work."""
import argparse
import datetime
import json
from pathlib import Path

root = Path(__file__).resolve().parent.parent
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('step')
parser.add_argument('status', choices=['in_progress', 'verified', 'blocked', 'planned'])
parser.add_argument('--note', required=True)
parser.add_argument('--evidence', action='append', default=[])
args = parser.parse_args()
path = root / 'docs/frontier-backlog.json'
data = json.loads(path.read_text())
steps = {s['id']: s for s in data['steps']}
if args.step not in steps:
    parser.error('Unknown step')
step = steps[args.step]
if args.status in ('in_progress', 'verified'):
    missing = [id for id in step['depends_on'] if steps[id]['status'] not in ('verified', 'documented')]
    if missing:
        parser.error('Unfinished dependencies: ' + ', '.join(missing))
if args.status == 'verified' and not args.evidence:
    parser.error('Verification requires evidence (test commands and outcome or reviewable files)')
now = datetime.datetime.now(datetime.timezone.utc).isoformat(timespec='seconds')
step.update(status=args.status, updated_at=now, progress_note=args.note)
step['evidence'] = args.evidence
if args.status == 'in_progress':
    step.setdefault('started_at', now)
data['status_meaning'].update(in_progress='Implementation underway; not counted complete', verified='Implemented and verified; see step evidence', blocked='Specific prerequisite or failing check prevents completion')
path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + '\n')
log = root / 'docs/frontier-progress.jsonl'
with log.open('a') as stream:
    stream.write(json.dumps({'at':now,'step':args.step,'status':args.status,'note':args.note,'evidence':args.evidence},ensure_ascii=False)+'\n')
counts = {state:sum(s['status']==state for s in steps.values()) for state in ('documented','verified','in_progress','planned','blocked','external')}
page = '# 구현 진행 기록\n\n'
page += f"최근 기록: {now}\n\n조사·설계 {counts['documented']}/8 · 신규 구현 검증 {counts['verified']}/72 · 진행 중 {counts['in_progress']} · 예정 {counts['planned']} · 차단 {counts['blocked']} · 외부 조건 {counts['external']}\n\n"
page += '전체 계획: [80단계](frontier-implementation-plan.md). 완료는 구현 후 검증 증거가 있는 경우만 센다. 실제 Mac·도메인·개인 키 검증은 fixture 성공과 구분한다.\n\n'
page += '| 시각 (UTC) | 단계 | 상태 | 결과·다음 작업 |\n| --- | --- | --- | --- |\n'
for line in log.read_text().splitlines():
    entry=json.loads(line)
    page += f"| {entry['at']} | {entry['step']} | {entry['status']} | {entry['note'].replace('|','/')} |\n"
(root/'docs/frontier-progress.md').write_text(page)
print(f"Recorded {args.step}={args.status}: {counts}")
