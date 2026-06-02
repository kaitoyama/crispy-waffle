# 09. 調査メモ（外部知見・標準・先行事例）

> **English summary**: Curated, sourced notes from the external landscape (as of
> 2026-06) that ground the design: agent identity standards, MCP authorization,
> OAuth token exchange / delegation, fine-grained authz (ReBAC/ABAC), workload
> identity (SPIFFE), capability/macaroon attenuation, durable execution &
> human-in-the-loop, agentic workflow graphs, MCP elicitation, and A2A. Each item
> notes *what it is* and *why it matters here*. This area is actively standardizing;
> treat links as living references.

> 注：本メモは設計の根拠となる外部情報の要約とリンク集です。AI エージェントの
> アイデンティティ／認可は標準化途上であり、内容は流動的です（調査時点 2026-06）。

## A. エージェント・アイデンティティの標準化動向

### A-1. OpenID Foundation「Identity Management for Agentic AI」白書（2025）
- **何か**：OIDF の AI Identity Management Community Group (AIIMCG) が公開。AI エージェントを
  IAM の一級主体として扱い、ライフサイクル・認証・認可・ガバナンスを管理する戦略を提言。
  既存標準（OAuth 2.0）の活用と、外部連携への MCP 採用を推奨。
- **なぜ重要か**：本設計の「エージェントを Actor として一級化」「委譲・縮小・監査」の裏づけ。
  現状ギャップ（スケーラブルなアクセス制御、エージェント中心 ID、委譲権限）の指摘が
  [05](./05-authz-identity.md) の前提。
- リンク: https://openid.net/new-whitepaper-tackles-ai-agent-identity-challenges/ ／
  arXiv: https://arxiv.org/abs/2510.25819

### A-2. 業界の総括（identity is the unsolved agentic problem）
- OAuth 2.1（2025 統合版）は公開クライアントに PKCE 必須・暗黙フロー廃止。
- OAuth 2.0 は 1 ホップ委譲は得意だが、**多段連鎖・クロスドメイン非同期・スコープと能力の
  対応**が未整備＝標準化の焦点。
- 規制動向：短命 OAuth/OIDC トークンの要求、明示認可なきエージェント間権限委譲の禁止
  （例：シンガポール CSA の Securing Agentic AI 補遺, 2025-10）。
- リンク: https://www.resilientcyber.io/p/identity-is-the-agentic-ai-problem ／
  https://prefactor.tech/blog/regulatory-standards-for-ai-agent-identity ／
  https://dev.to/arcade/how-to-manage-multi-user-ai-agent-authentication-and-authorization-in-2026-oauth-21-oidc-and-2943

## B. MCP（Model Context Protocol）認可

### B-1. MCP Authorization 仕様
- **何か**：MCP サーバを **OAuth 2.1 リソースサーバ**と位置づけ、トークン検証に徹し、認可
  サーバは別に置く。**Protected Resource Metadata (RFC 9728)** で認可サーバ所在を広告、
  **Resource Indicators (RFC 8707)** でトークンを対象資源に限定。
- **なぜ重要か**：[04 §7](./04-tool-integration.md) の `provider_binding`、[05](./05-authz-identity.md)
  のスコープ最小化の具体路線。「ツール実体は認可を外部に委ねる」設計の裏づけ。
- リンク: https://modelcontextprotocol.io/specification/draft/basic/authorization ／
  https://auth0.com/blog/mcp-specs-update-all-about-auth/ ／
  https://den.dev/blog/mcp-november-authorization-spec/ ／
  https://www.descope.com/blog/post/mcp-auth-spec

### B-2. MCP Elicitation
- **何か**：2025-06-18 仕様で導入（draft）。MCP サーバが利用者に追加入力や承認/却下を求める。
  「本当に削除しますか？」型の確認に使い、No/キャンセルを必ず提供。2025-11-25 で URL モード追加。
- **なぜ重要か**：[07](./07-human-in-the-loop.md) の「不可逆操作直前の Yes/No 確認」の実体候補。
- リンク: https://thenewstack.io/how-elicitation-in-mcp-brings-human-in-the-loop-to-ai-tools/ ／
  https://github.blog/ai-and-ml/github-copilot/building-smarter-interactions-with-mcp-elicitation-from-clunky-tool-calls-to-seamless-user-experiences/ ／
  https://gofastmcp.com/servers/elicitation

## C. 委譲・トークン交換

### C-1. OAuth 2.0 Token Exchange（RFC 8693）
- **何か**：あるトークンを別トークンに交換する標準（On-Behalf-Of）。`subject_token`（誰の代わり）
  と `actor_token`（誰が代行）を区別。`act` 入れ子クレームで委譲連鎖を表現、`may_act` で
  「誰が交換してよいか」を事前認可。
- **注意**：`act` は連鎖の**足跡（情報）**で、それ自体は強制力を持たない。
- **なぜ重要か**：[05 §3](./05-authz-identity.md) の委譲・縮小の中核。「記録だけでなく
  認可ゲートで強制する」という本設計の方針の根拠。
- リンク: https://www.rfc-editor.org/info/rfc8693/ ／
  https://workos.com/blog/oauth-multi-hop-delegation-ai-agents ／
  https://zitadel.com/docs/guides/integrate/token-exchange

### C-2. Identity Assertion JWT Authorization Grant / Cross-App Access (XAA)
- **何か**：IETF OAuth WG の Internet-Draft（2026）。企業 IdP が ID トークン（OIDC）を仲介し、
  ユーザー手動同意をトークン交換（RFC 8693 + JWT Bearer RFC 7523）に置換。アプリ間・
  エージェント↔アプリ間接続を IdP が統制し、トークンに**人間とエージェント両方の文脈**を載せる。
- **なぜ重要か**：[05 §6](./05-authz-identity.md) の組織境界を越えた委譲＋監査の有力路線。
- リンク: https://datatracker.ietf.org/doc/draft-ietf-oauth-identity-assertion-authz-grant/ ／
  https://oauth.net/cross-app-access/ ／
  https://www.okta.com/identity-101/cross-app-access-securing-ai-agent-and-app-to-app-connections/

## D. 細粒度認可（ReBAC / ABAC）

### D-1. OpenFGA / Google Zanzibar（ReBAC）
- **何か**：`(object, relation, user)` タプルで関係を表し、関係に基づき判定する ReBAC。
  OpenFGA は CNCF/Okta、Zanzibar は Google の全社認可基盤の論文由来。RBAC/ABAC も包含。
- **なぜ重要か**：[05 §4.2](./05-authz-identity.md) の「この案件の承認者か」「自部署の経費か」
  判定の実装路線。エージェント/RAG/MCP の認可にも適用される事例あり。
- リンク: https://openfga.dev/docs/authorization-concepts ／
  https://auth0.com/blog/rebac-abac-openfga-cedar/

### D-2. ReBAC × ABAC × Cedar
- **何か**：関係（ReBAC）と属性条件（ABAC, Cedar 等）を併用するのが実務的。
- **なぜ重要か**：[05 §4](./05-authz-identity.md) の「ロール＋ReBAC＋ABAC の三段」と PEP/PDP 分離の根拠。
- リンク: https://auth0.com/blog/rebac-abac-openfga-cedar/ ／
  https://www.couchbase.com/blog/securing-agentic-rag-pipelines/

## E. ワークロード・アイデンティティ（非人間 ID）

### E-1. SPIFFE / SPIRE
- **何か**：ワークロードに短命・自動ローテーションの検証可能 ID（SVID）を付与する標準。
  人ではなくワークロードに紐づくため AI エージェント等の非人間主体に適する。
  「SVID を取得 → スコープつきクラウド資格情報へ交換」のパターン。
- **なぜ重要か**：[05 §2.2](./05-authz-identity.md) のエージェント NHI の一路線。静的キー使い回しの回避。
- リンク: https://www.hashicorp.com/en/blog/spiffe-securing-the-identity-of-agentic-ai-and-non-human-actors ／
  https://riptides.io/blog-post/spiffe-meets-oauth2-current-landscape-for-secure-workload-identity-in-the-agentic-ai-era/

## F. ケイパビリティ／マカロン（縮小可能な権限）

### F-1. Capability-based security & Macaroons
- **何か**：偽造不能なトークンが操作権を内包。**attenuation**＝親より狭い権限を派生。
  マカロンは伝播時に **caveat（制約）を追記**して縮小、HMAC 連鎖で完全性、検査可能。
- **なぜ重要か**：[05 §5](./05-authz-identity.md) のケイパビリティ設計、「各ホップで権限が
  増えない」「失効伝播」の理論的支柱。
- リンク: https://freecontent.manning.com/capability-based-security-and-macaroons/ ／
  https://www.okta.com/blog/ai/agent-security-delegation-chain/

## G. 永続実行と Human-in-the-loop

### G-1. Durable Execution（Temporal 等）と承認待ち
- **何か**：ワークフロー状態を永続化し、`wait_condition` 等で**待機中は計算資源 0**。シグナルや
  タイムアウトで再開。承認の押下を耐久保存し、クラッシュ後も「承認済み」で再開。
- **なぜ重要か**：[03 §3](./03-workflow-engine.md) の中断/再開と [07 §3.2](./07-human-in-the-loop.md)
  の承認の耐久性の概念的裏づけ（※本設計は特定エンジン非依存）。
- リンク: https://temporal.io/blog/human-in-the-loop-approvals ／
  https://learn.temporal.io/tutorials/ai/building-durable-ai-applications/human-in-the-loop/ ／
  https://truto.one/blog/implementing-human-in-the-loop-approval-workflows-for-consequential-saas-api-actions/

## H. エージェント・ワークフローのグラフ表現

### H-1. グラフ／DAG ベースのオーケストレーション
- **何か**：LLM ワークフローを有向グラフ（ノード＝ツール/LLM 呼び出し、エッジ＝遷移）で表す
  runtime（LangGraph 等）と durable execution（Temporal 等）が本番標準化。タスク分解は DAG。
- **なぜ重要か**：[03 §9](./03-workflow-engine.md) の状態機械/グラフ表現の位置づけ。
- リンク: https://aiworkflowlab.dev/article/ai-workflow-orchestration-in-production-building-durable-agent-pipelines-with-langgraph-and-temporal ／
  https://www.diagrid.io/ai-orchestration

## I. エージェント間連携

### I-1. A2A（Agent2Agent）プロトコル
- **何か**：Google 主導（2025-04）。異ベンダのエージェントが相互発見・タスク委譲・協調。
  **Agent Card**（JSON の「名刺」）で ID・エンドポイント・**能力**を広告し、レジストリで発見。
  HTTP / SSE / JSON-RPC 2.0。
- **なぜ重要か**：[04 §5](./04-tool-integration.md) の「能力の発見」や、将来の外部エージェント
  連携時の標準候補。
- リンク: https://a2a-protocol.org/latest/specification/ ／
  https://developers.googleblog.com/en/a2a-a-new-era-of-agent-interoperability/

## J. 設計への含意（横断まとめ）

| 外部知見 | 本設計への反映先 |
|---------|----------------|
| OIDF 白書 / OAuth2.1 | エージェントの一級化・委譲（[05](./05-authz-identity.md)） |
| MCP Authorization | ツールの provider_binding・スコープ最小化（[04](./04-tool-integration.md)） |
| MCP Elicitation | 不可逆操作前の確認（[07](./07-human-in-the-loop.md)） |
| Token Exchange / XAA | 委譲連鎖・クロスアプリ（[05](./05-authz-identity.md)） |
| OpenFGA/Zanzibar・Cedar | ReBAC+ABAC の判定（[05](./05-authz-identity.md)） |
| SPIFFE | エージェント非人間 ID（[05](./05-authz-identity.md)） |
| Macaroons | ケイパビリティの縮小（[05](./05-authz-identity.md)） |
| Durable Execution | 中断/再開・承認の耐久性（[03](./03-workflow-engine.md), [07](./07-human-in-the-loop.md)） |
| LangGraph/DAG | ワークフローのグラフ表現（[03](./03-workflow-engine.md)） |
| A2A Agent Card | 能力の発見（[04](./04-tool-integration.md)） |
