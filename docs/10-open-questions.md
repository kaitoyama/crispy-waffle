# 10. 未決事項・ユーザーへの確認リスト

> **English summary**: Open decisions surfaced while writing the design set, grouped
> by plane and priority. These are deliberately *not* resolved yet — they need
> product/owner input before we lock a technology stack or build. Each item links to
> where it arose.

各ドキュメントを書く中で浮かんだ未決事項を集約します。これらは**わざと未確定**にしてあり、
技術選定や実装に入る前にユーザー（オーナー）と詰めるべき論点です。

## 優先度: 高（プロダクトの輪郭を決める）

| # | 論点 | 背景・選択肢 | 出所 |
|---|------|------------|------|
| H1 | **最初に作る対象業務はどれか** | 経費精算を縦に 1 本通すか、複数業務の横断基盤を先に作るか。MVP の形を決める | [00](./00-vision.md), [08](./08-reference-workflows.md) |
| H2 | **利用組織の想定規模・文脈** | サークル／小規模団体が主か、企業も視野か。規模で認可・監査の要求が変わる | [05](./05-authz-identity.md) |
| H3 | **「チャット」の位置づけ** | 入力手段の一つに留めるか、各タスクにスレッドを併設するか | [00](./00-vision.md) |
| H4 | **エージェントの自律度** | どこまで自動で進め、どこで必ず人を挟むか（既定の HITL ポリシー） | [07](./07-human-in-the-loop.md) |

## 優先度: 中（モデルの確定に必要）

| # | 論点 | 背景・選択肢 | 出所 |
|---|------|------------|------|
| M1 | `human_gate` の細分 | 「承認（他者）」と「実行トリガ（本人の"精算する"押下）」を別種別にするか | [08 §4](./08-reference-workflows.md) |
| M2 | 規定（領収書要否等）の置き場所 | TaskType か、横断ポリシー（ABAC）か | [08 §4](./08-reference-workflows.md), [05 §4.3](./05-authz-identity.md) |
| M3 | 副作用クラスの粒度 | `irreversible` を金銭／対外送信／その他に細分するか | [04 §3](./04-tool-integration.md) |
| M4 | リンク型の最終集合 | subtask_of/blocks/relates_to/triggered_by/approval_for/supersedes で十分か | [02 §4](./02-domain-model.md) |
| M5 | タスクと実行の基数 | 1:1 を基本とするが、再試行・複数フロー時の扱い | [02 §12](./02-domain-model.md) |
| M6 | スコープ軸の標準セット | 金額／対象／時間／データ範囲… の確定 | [04 §4](./04-tool-integration.md) |
| M7 | 定期タスクの境界の持ち方 | `last_boundary` を定義ごと 1 つか、対象（人・勘定）ごとか | [06 §9](./06-recurring-tasks.md) |

## 優先度: 低（実装段階で決めてよい・技術選定）

| # | 論点 | 背景・選択肢 | 出所 |
|---|------|------------|------|
| L1 | 認可エンジン | 内製か外部（OpenFGA / Cedar 等）か | [05 §10](./05-authz-identity.md) |
| L2 | ケイパビリティ実装 | トークン内包（マカロン的）か中央 PDP 問い合わせか | [05 §10](./05-authz-identity.md) |
| L3 | エージェント NHI | SPIFFE 路線か IdP 発行 NHI か | [05 §10](./05-authz-identity.md) |
| L4 | ツール接続 | MCP を第一実装にするか、抽象カタログを先に固めるか | [04 §9](./04-tool-integration.md) |
| L5 | ワークフロー実行基盤 | durable execution エンジンを使うか内製か | [03](./03-workflow-engine.md) |
| L6 | スキーマ記述手段 | context/tool スキーマを JSON Schema 相当で統一するか | [02 §12](./02-domain-model.md), [04 §9](./04-tool-integration.md) |
| L7 | スケジュール記述 | cron / 暦ルール / 自然言語、TZ・営業日・祝日 | [06 §9](./06-recurring-tasks.md) |

## ユーザーに今すぐ確認したいこと（次アクション）

設計をさらに進めるために、特に **H1〜H4** の方向づけがほしいです。

1. **H1**: 最初の縦串は「交通費の精算フロー」で良いか。
2. **H2**: 主たる利用者像（サークル／小規模団体 を想定して良いか）。
3. **H4**: 金銭を伴う操作は「常に人間の確認＋承認を必須」とする保守的な既定で良いか。

> これらが決まれば、次の MD として「MVP スコープ定義」と「初回縦串の詳細設計」を起こせます。
