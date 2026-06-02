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
| H1 | **最初に作る対象業務はどれか** | 経費精算を縦に 1 本通すか、複数業務の横断基盤を先に作るか。MVP の形を決める。経費は Jomon と重なるため N1 と連動 | [00](./00-vision.md), [08](./08-reference-workflows.md), [11](./11-trap-ecosystem.md) |
| ~~H2~~ | ~~利用組織の想定規模・文脈~~ | ✅ **解決**：traP（東京科学大学デジタル創作同好会、約250名）＋ traP-OSS 利用者。詳細は [11](./11-trap-ecosystem.md) | [11](./11-trap-ecosystem.md) |
| H3 | **「チャット」の位置づけ** | 入力手段の一つに留めるか、各タスクにスレッドを併設するか。traP では traQ がチャット基盤 | [00](./00-vision.md), [11](./11-trap-ecosystem.md) |
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

## traP 接地で新たに加わった論点（[11](./11-trap-ecosystem.md) より）

| # | 論点 | 背景・選択肢 | 出所 |
|---|------|------------|------|
| N1 | **Jomon は連携か置換か** | 推奨は「ツール連携」（Jomon v2 API を叩く）。置換は移行コスト大。経費 MVP の形を左右（H1 と連動） | [11 §5](./11-trap-ecosystem.md) |
| N2 | **スタンプ承認の安全策** | 金銭を伴う不可逆操作で、スタンプ 1 つの承認は誤操作リスク。明示確認の併用要否 | [11 §4](./11-trap-ecosystem.md) |
| N3 | **エージェントの実装形態** | traQ BOT か、OAuth2 クライアントか、両方か | [11 §3](./11-trap-ecosystem.md) |
| N4 | **既存レコードとの同一性** | 本システムの「タスク」と Jomon/knoQ の既存レコードのリンク・重複回避 | [11 §9](./11-trap-ecosystem.md) |

## 認可平面で接地により方針が固まった点

- 人間の IdP：**traQ OAuth2（auth code + PKCE）** を採用（L レベルの ID 基盤論点は実質解決）。
- エージェント NHI（L3）：当面 **traQ OAuth2 クライアント**で足りる。SPIFFE 等は将来拡張。

## ユーザーに今すぐ確認したいこと（次アクション）

利用者像（H2）が traP に確定したことで、次は **H1・N1・H4** の方向づけがほしいです。

1. **H1 / N1**: 最初の縦串は「交通費の精算フロー」で良いか。その際 **Jomon は連携（ツール）**で良いか
   （置換ではなく、Jomon API を叩いてエージェント＋承認連鎖を上に載せる）。
2. **N3**: エージェントは traQ BOT として実装する想定で良いか。
3. **H4 / N2**: 金銭を伴う操作は「常に人間の確認＋承認を必須」とし、スタンプ承認には明示確認を
   併用する保守的な既定で良いか。

> これらが決まれば、次の MD として「MVP スコープ定義」と「Jomon 連携を前提とした
> 交通費精算の詳細設計」を起こせます。
