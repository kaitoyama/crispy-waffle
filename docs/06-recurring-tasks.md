# 06. 定期タスク（スケジューリングと区間処理）

> **English summary**: Recurring task definitions materialize task instances on a
> schedule (e.g. 1st & 15th monthly). The key semantic is **interval processing**:
> each run covers the period **[last_boundary, this_run)** so it sweeps up everything
> since the previous run — driven from the event ledger, not from wall-clock guesses.
> Defines schedule types, catch-up policies for missed runs, idempotent
> materialization, and how a recurring payment ("pay ¥20,000 on the 1st and 15th")
> maps onto the model.

## 1. なぜ定期タスクが必要か

ビジョン（[00](./00-vision.md)）の課題 C：チャットは一回性で、繰り返しを表せない。
「毎月 1 日と 15 日に支払う」「前回処理した日から今日までの分をまとめて精算する」といった
**定期・区間**の業務を、第一級の仕組みとして持つ。

## 2. RecurringTaskDefinition の再掲（[02 §9](./02-domain-model.md)）

| フィールド | 意味 |
|-----------|------|
| `schedule` | いつ発火するか（例：毎月 1 日・15 日 09:00） |
| `interval_semantics` | 区間の意味（`since_last_run` / `fixed_window` / `point_in_time`） |
| `catch_up_policy` | 取りこぼし時の挙動 |
| `last_boundary` | 最後に処理した区間の終端（次回の起点） |
| `task_type` / `template_context` | 生成するタスクの型と初期データ |

## 3. 区間処理（Interval Processing）— 本章の核心

定期タスクの本質は「いつ動くか」よりも「**どの範囲を対象にするか**」。

### 3.1 `since_last_run`（前回実行〜今回）

最も重要な意味論。各実行は **`[last_boundary, this_run)`** の半開区間を対象に、その間に
発生した対象（経費・支払い・イベント）を**すべて吸い上げて**処理する。

```
前回実行 last_boundary = 5/15
今回実行 this_run      = 6/1
  → 対象期間 = [5/15, 6/1)
  → この期間に積み上がった経費・案件を集約して処理
処理後 last_boundary ← 6/1
```

これがビジョンの「前回生産した日から今回生産する日までの間が対象になる」の実体。
**境界はイベント台帳（[02 §8](./02-domain-model.md)）から計算**し、壁時計の当て推量に頼らない。

### 3.2 `fixed_window`（暦に固定）

「当月 1 日〜末日」のように暦で固定した窓。`since_last_run` と違い、取りこぼしても窓は
ずれない。月次レポートなどに向く。

### 3.3 `point_in_time`（時点アクション）

範囲集約ではなく「その時点で 1 アクション」。例：固定額の定例支払い（後述 §5）。

## 4. スケジュールと取りこぼし（Catch-up）

システム停止や遅延で発火が漏れることがある。`catch_up_policy` で挙動を決める。

| ポリシー | 挙動 | 向く用途 |
|---------|------|---------|
| `coalesce`（合流） | 漏れた回をまとめて 1 回で処理 | 区間集約（`since_last_run`）。範囲を広げれば 1 回で吸収できる |
| `each`（各回生成） | 漏れた各回を個別に生成 | 各回が独立した意味を持つ場合 |
| `skip`（飛ばす） | 漏れた回は捨て、次の正規回から | 時点アクションで重複が危険な場合 |

> `since_last_run` × `coalesce` の組み合わせが、区間処理の取りこぼしに最も素直。
> 境界を「前回成功時点」に保つ限り、いつ再開しても範囲が連続する。

## 5. 例：固定額の定例支払い（毎月 1 日・15 日に 2 万円）

ビジョンの「定例で 2 万円を出すタスク」のモデル化。

```yaml
RecurringTaskDefinition:
  key: payroll.stipend.tanaka
  schedule: 毎月 1日・15日 09:00
  interval_semantics: point_in_time   # 固定額の時点アクション
  catch_up_policy: skip               # 二重支払い防止（時点は飛ばす）
  task_type: payment.fixed
  template_context:
    amount: 20000
    payee: user:tanaka
    account: circle-main
```

各発火で生成されるタスクは、通常の承認・実行フロー（[03](./03-workflow-engine.md),
[08](./08-reference-workflows.md)）に乗る。支払いは `irreversible_or_monetary`（[04 §3](./04-tool-integration.md)）
なので、**冪等キー**（例：`payroll.stipend.tanaka:2026-06-01`）で二重支払いを必ず防ぐ。

## 6. 例：区間集約の精算（前回処理日〜今日の経費をまとめて）

```yaml
RecurringTaskDefinition:
  key: expense.settle.monthly
  schedule: 毎月 1日 10:00
  interval_semantics: since_last_run   # 区間集約
  catch_up_policy: coalesce
  task_type: expense.settlement.batch
```

発火時のタスクは：
1. `[last_boundary, now)` の対象案件を**吸い上げる**（[03 §6](./03-workflow-engine.md) のファンアウト）。
2. 各案件が経理ラインを越えていないか・要件を満たすか判定。
3. 経費として正当なものだけを、順番に支払い（精算）処理へ。
4. 成功したら `last_boundary ← now`。

## 7. 冪等な materialization（生成の安全性）

- 同じスケジュール時点から**二重にタスクを生成しない**。生成キー（定義キー＋発火時点）で冪等化。
- 生成は台帳に `recurring.materialized` イベントとして残し、`triggered_by` リンクで定義と結ぶ。
- `last_boundary` の更新は**処理成功イベントに連動**させ、失敗時は境界を進めない
  （次回が同じ区間を再処理 ＝ at-least-once ＋ 冪等で実質 exactly-once）。

## 8. スケジュールとシグナルの関係

定期発火は、[03 §3](./03-workflow-engine.md) の「タイマーシグナル」の一種。眠っている実行
（締め待ち等）を、時刻トリガで起こす。承認シグナル・外部完了シグナルと同じ再開機構に乗る。

## 9. 未決事項

- スケジュール記述の表現（cron 式 / 暦ルール / 自然言語）。
- `last_boundary` を定義ごとに 1 つにするか、対象（人・勘定）ごとに持つか。
- タイムゾーン・営業日・祝日の扱い。

→ [10-open-questions.md](./10-open-questions.md)
