# 11. traP エコシステムへの着地

> **English summary**: Grounds the (until-now abstract) design in the concrete target
> environment: the **traP** OSS ecosystem (Digital Creators Club at Institute of
> Science Tokyo / former Tokyo Tech, ~250 members). Key findings: **traQ** is the
> club's chat platform **and a built-in OAuth2 authorization server** (auth-code +
> PKCE) that already acts as the SSO/identity provider for every traP service —
> so human identity ([05]) is essentially solved. **Jomon** is traP's existing
> accounting/expense system (requests → approval → transactions, receipts), i.e. our
> anchor example already exists. The product is best framed as an **LLM-agent
> orchestration layer over the existing traP services** (traQ = identity + chat
> surface; Jomon/knoQ/anke-to/booQ/NeoShowcase = tools in the catalog).

このドキュメントは、[00](./00-vision.md)〜[10](./10-open-questions.md) の技術非依存な設計を、
**実際の対象環境（traP の OSS 群）** に接地させます。これにより抽象論が一気に具体化します。

## 1. 対象ユーザー：traP とそのエコシステム

- **traP（東京科学大学デジタル創作同好会）**：旧 東京工業大学。約 250 名規模の学生サークル。
- 多数の内製 OSS を運用し、すべて GitHub `traPtitech` で公開・自前ホスト（`*.trap.jp`）。
- 部員は日常的にこれらのサービスを横断利用している。

→ つまり本プロダクトは、**ゼロから ID 基盤や業務システムを作るのではなく、既存の traP
サービス群の上に「エージェントによるオーケストレーション層」を載せる**のが自然な立ち位置。

## 2. 既存サービスと三層モデルの対応

[README](./README.md) の三層に、既存 traP サービスを当てはめると次のようになる。

| 平面 | 役割 | 既存 traP サービス／資産 |
|------|------|------------------------|
| 認可平面 | 人間の認証・SSO | **traQ の OAuth2 認可サーバー**（後述 §3） |
| 作業平面 | 既存の業務データ | **Jomon**（会計）、knoQ（イベント/部屋）等が持つ申請・予約 |
| 実行平面 | 通知・対話・承認の場 | **traQ**（チャット、BOT、スタンプ） |
| ツール | 外部作用の対象 | Jomon / knoQ / anke-to / booQ / NeoShowcase 等の REST API |

## 3. traQ = アイデンティティ・プロバイダ（[05] の具体的着地）

**最重要の発見**：traQ は単なるチャットではなく、**部内製の OAuth 2.0 認可サーバー**を内蔵する。

- エンドポイント：`https://q.trap.jp/api/v3/oauth2/authorize`、`.../oauth2/token`
- **認可コードフロー ＋ PKCE** に対応（[05 §2.1](./05-authz-identity.md) の OAuth2.1 基線と整合）。
- Jomon・knoQ など各サービスは **traQ アカウントでログイン**（traQ クライアント ID/secret を設定）。
- 部員は誰でもクライアントアプリを登録できる。

### これが意味すること

[05](./05-authz-identity.md) で「人間は OIDC で認証」と述べた部分は、traP では **traQ OAuth2 が
既に事実上の IdP** として機能している、と具体化できる。新たな ID 基盤を作る必要はない。

```mermaid
graph LR
    User[部員] -->|traQ OAuth2 (auth code + PKCE)| Auth[traQ 認可サーバー]
    Auth -->|アクセストークン| Waffle[crispy-waffle]
    Waffle -->|ユーザーの委譲権限で| Tools[Jomon / knoQ / ...]
```

### エージェント（非人間 ID）の現実的な落とし所

[05 §2.2](./05-authz-identity.md) の「エージェントの非人間アイデンティティ」は、traP 文脈では：

- **エージェントは traQ の BOT／OAuth2 クライアントとして登録**し、固有の身元を持つ。
- ユーザーから委譲を受けた範囲で、**ユーザーの traQ トークンに紐づくスコープ**で各サービスを叩く。
- 委譲・縮小（attenuation, [05 §3](./05-authz-identity.md)）は、traQ が発行するトークンのスコープと、
  本システムの認可ゲート（[04 §6](./04-tool-integration.md)）の積で実現する。
- SPIFFE 等の重厚な NHI 基盤（[05 §2.2](./05-authz-identity.md)）は当面オーバースペック。
  まずは「traQ OAuth2 クライアント＋本システム内のケイパビリティ」で十分実用的。

> 標準動向（XAA / Token Exchange、[09](./09-research-notes.md)）は将来の拡張余地として保持しつつ、
> **第一実装は traQ OAuth2 を素直に使う**のが現実解。

## 4. traQ = 対話・通知・承認の場（[07] の具体的着地）

traQ は BOT API・Webhook・**スタンプ（リアクション）**を持つ。これらが HITL（[07](./07-human-in-the-loop.md)）
の実体になる。

| HITL 概念（[07]） | traQ での実体 |
|------------------|--------------|
| 確認 / Elicitation（Yes/No） | BOT がメッセージを投稿し、ユーザーが返信／スタンプで応答 |
| 承認ゲート | 承認依頼を担当チャンネルに投稿し、**承認スタンプ**で意思表示 → ApprovalRecord 生成 |
| 通知 | 状態変化・期限・差し戻しを traQ メッセージで通知 |
| 最小文脈の承認面 | traQ メッセージ＋埋め込み（申請の要約だけを提示） |

traP 文化では**スタンプによる軽量な合意形成**が日常的であり、「承認スタンプで承認」は
極めて自然な UX。go-traq 等のクライアントライブラリ（[awesome-trap]）で実装も容易。

> 注：スタンプは軽量で便利だが、金銭を伴う不可逆操作の最終承認には、誤操作防止のため
> 明示的な確認ステップを併用するか検討（[10](./10-open-questions.md) に追記）。

## 5. Jomon = 既存の**部費**会計フロー（[08] の現実版）

**Jomon は traP の「部費（部の公式予算）」専用の会計システム**であり、本ドキュメントの
経費精算例（[08](./08-reference-workflows.md)）は **既に存在する業務**である。
逆に言うと、**部費外のお金（例：合宿の参加費）は Jomon の対象外**で、そちらは rucQ など別系統が扱う
（[12 §A-0/B-1](./12-cross-service-usecases.md)）。

- Go ＋ MariaDB、REST API（Jomon v2 API、`docs/swagger.yaml` を精読）、UI 分離（Jomon-UI）。
- **traQ で認証（OAuth PKCE）**し、**traQ へ通知**する（§3・§4 の構図に乗っている）。
- 実ドメイン（検証済み）：
  - **Application（申請）**：`title`/`content`/`targets[]`(振込先＋金額)/`tags[]`/`partition`(予算区分)。
  - **Status**：`pending_review → change_requested → approved → payment_finished`（or `rejected`）。
  - **ApplicationTarget（振込対象）**：`amount`/`target`/`paid_at`（支払い完了で payment_finished）。
  - **Partition/PartitionGroup**：`budget` を持つ階層的**予算区分**。
  - **AccountManager（会計担当）**、Comment、Tag、File（領収書）。
- ⚠️ **重要差分**：現状の Jomon は**単一 Application のライフサイクル**で、ユーザー構想の
  「事前申請 → 後日 精算」という**二段階とは構造が異なる**（[12 §A-0](./12-cross-service-usecases.md) J1）。

### 設計上の重要な分岐点

経費精算について、本プロダクトの立ち位置は二択：

1. **Jomon を「ツール」として叩く**（[04](./04-tool-integration.md) のツールカタログに Jomon API を登録）。
   エージェントが申請作成・承認・取引記録を Jomon 経由で行う。**既存資産を活かす穏当な路線**。
2. **Jomon の機能の一部を本システムで再実装／置換**する。連鎖・定期・エージェント実行を
   ネイティブに持てるが、二重管理・移行のコストが高い。

→ **推奨は (1)**：まず Jomon を外部ツールとして連携し、本プロダクトは「エージェント＋ワークフロー＋
承認の連鎖」という Jomon に無い価値を上に載せる。[10](./10-open-questions.md) の論点として明記。

## 6. その他サービス＝ツールカタログの初期メンバー（[04]）

| サービス | 概要 | ツール化の例 |
|---------|------|-------------|
| **knoQ** | 進捗部屋・イベント／部屋予約（iCal 対応） | 部屋の空き照会(read_only)、予約作成(reversible) |
| **anke-to** | アンケート／投票 | アンケート作成・集計取得 |
| **booQ** | 物品貸出管理 | 在庫照会、貸出申請 |
| **NeoShowcase** | 内製 PaaS（Docker/K8s デプロイ） | デプロイ状態照会、再デプロイ（要承認） |
| **traPortfolio / showcase** | 部員のポートフォリオ／作品 | プロフィール参照 |

各サービスは REST API ＋ traQ 認証で統一されているため、[04 §2](./04-tool-integration.md) の
ツール定義（スキーマ・副作用クラス・スコープ）に素直に載せられる。**副作用クラスの判定**が
そのまま「読み取り照会は自由／予約・申請は確認／デプロイ・支払いは承認」に対応する。

## 7. プロダクトの再定義（接地後の一文）

> **crispy-waffle は、traP の既存 OSS 群（traQ・Jomon・knoQ・anke-to・booQ・NeoShowcase…）の
> 上に載る「LLM エージェントによるタスク／ワークフロー・オーケストレーション層」である。
> 人間の認証は traQ OAuth2 に委ね、対話・通知・承認は traQ 上で行い、各サービスは
> ツールカタログ経由でエージェントが権限つきで叩く。チャット（traQ）は入力・承認の手段の
> 一つであり、本体は永続的なタスクの台帳と、権限つき実行・承認の連鎖である。**

## 8. この接地が解消／変更する論点

| 元の論点（[10]） | 接地後 |
|----------------|--------|
| H2 利用者像 | **解決**：traP（学生サークル、約250名）＋ traP-OSS 利用者 |
| L3 エージェント NHI | **方針確定寄り**：当面 traQ OAuth2 クライアントで足りる。SPIFFE は将来 |
| 人間の IdP（[05]） | **解決**：traQ OAuth2（auth code + PKCE） |
| HITL チャネル（[07]） | **解決寄り**：traQ メッセージ＋スタンプ |
| 経費フローの位置づけ | **新論点**：Jomon を「ツール連携」するか「置換」するか（推奨：連携） |

## 9. 新たに加わる未決事項（[10] に反映）

- **N1**：Jomon は連携（ツール）か置換か。連携なら Jomon v2 API のどこまでを叩くか。
- **N2**：承認をスタンプで行う際の安全策（金銭操作での誤操作防止）。
- **N3**：エージェントを traQ BOT として実装するか、OAuth2 クライアントとして実装するか、両方か。
- **N4**：本システムが扱う「タスク」と、Jomon/knoQ が持つ既存レコードの**同一性・リンク**の取り方。

## 参考リンク

- traPtitech GitHub: https://github.com/traPtitech
- traQ（チャット＋OAuth2 認可サーバー）: https://github.com/traPtitech/traQ
- 「traP には部内製の OAuth2.0 認可サーバーがあります」: https://trap.jp/post/1876/
- Jomon（会計支援システム）: https://github.com/traPtitech/Jomon
- knoQ（イベント/部屋）: https://github.com/traPtitech/knoQ
- awesome-trap（ライブラリ一覧）: https://github.com/traPtitech/awesome-trap
- traQ API（Swagger）: https://apis.trap.jp/
