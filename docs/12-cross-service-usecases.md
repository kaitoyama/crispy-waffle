# 12. サービス別ユースケースと横断連携

> **English summary**: Explores the *use-case* surface across the whole traP service
> catalog and — crucially — **cross-service composite workflows**, which is where an
> agent orchestration layer adds value no single existing service provides.
> Part A: per-service tasks/tools (knoQ, booQ, anke-to, NeoShowcase, rucQ,
> trap-collection, traPortfolio, traPortal, Qall) with side-effect classes.
> Part B: cross-service links where the orchestration layer genuinely adds value.
> **Correction (v2):** rucQ is NOT a thin hub to be assembled from anke-to/knoQ/Jomon —
> it is a near-complete camp-ops system of its own (registration, deadline-bound
> pre-questions, payment/collection, rooming, roll-calls, schedule, guidebook). So the
> real value is connecting rucQ's *income* side to Jomon's *expense* side and adding
> proactive chasing/approval — not rebuilding it. Other genuine composites: event→expense,
> equipment purchase lifecycle. Part C: not-yet-captured
> cross-cutting use cases (deadlines/reminders, onboarding, role handover, audits).
> Part D: design implications (cross-service task links, unified catalog, traQ-scoped
> access). This doc widens [04] and [08] beyond the accounting anchor.

このドキュメントは、[11](./11-trap-ecosystem.md) で把握した traP サービス群について、
「**どんなタスクが載るか**」「**サービスをまたいで何ができるか**」「**まだ拾えていない用途**」を
広く掘ります。設計の中心命題——*単一サービスでは完結しない仕事をエージェントが連鎖でつなぐ*
——を具体例で裏づけるのが狙いです。

## traP サービス全体マップ（接地の確定版）

| サービス | 何をするか | 本システムでの主な役割 |
|---------|-----------|----------------------|
| **traQ** | チャット＋OAuth2 認可サーバー＋BOT＋スタンプ＋**Qall**(通話, LiveKit) | 認証・対話・通知・承認の場（[11](./11-trap-ecosystem.md)） |
| **traPortal** | 部員情報管理・招待コード | 認可平面（ロール・在籍）の参照元 |
| **knoQ** | 進捗部屋・イベント・部屋予約（iCal） | ツール：予約・空き照会 |
| **Jomon** | **部費（部の公式予算）専用**の会計。Application を `pending_review→approved→payment_finished` で回し、partition(予算区分)・account_manager(会計担当)・振込対象・領収書を持つ。**合宿等の参加費は対象外** | ツール：部費の申請・承認・振込（[08](./08-reference-workflows.md)） |
| **anke-to** | 部内アンケート・投票 | ツール：意思決定・日程調整の入力 |
| **booQ** | 備品・書籍の貸出管理（返却期限・カレンダー） | ツール：在庫照会・貸出・返却 |
| **NeoShowcase** | 内製 PaaS（500+ アプリ、自動ビルド/デプロイ） | ツール：デプロイ・状態照会 |
| **traPortfolio** | 部員の活動ポートフォリオ | ツール：活動・実績の参照（主に read_only） |
| **rucQ** | **合宿運営システム**（登録・締切つき事前質問・集金・部屋割り・点呼・しおり・スケジュールを内製） | ほぼ自己完結。本システムは外側の催促・承認・対 Jomon 連携を担う（B-1） |
| **trap-collection** | 内製ゲームの管理・配布 | ツール：作品登録・公開 |

---

# Part A. サービス別ユースケースとツール化

各サービスを [04](./04-tool-integration.md) のツール（副作用クラス付き）として捉え、
そこに載るタスク・承認・定期パターンを洗い出す。

> ⚠️ **確度について**：rucQ（[A-5](#a-5-rucq合宿運営)）は `openapi.yaml` ＋ `model/*.go` を
> 実際に読んで記述しており確度が高い。一方、他サービス（knoQ・booQ・anke-to・NeoShowcase 等）の
> 機能記述は紹介記事・README 等の軽い情報に基づく**暫定**で、rucQ で起きたような取り違えの
> 可能性がある。MVP 前に各サービスの OpenAPI を rucQ 同様に精読して確定する（[10 U2](./10-open-questions.md)）。

## A-0. Jomon（部費会計）※実機能を検証済み

> `docs/swagger.yaml`（v2）を精読。Jomon は **部費（部の公式予算）専用**の申請・承認・振込システム。
> **合宿の参加費など部費外のお金は対象外**（[B-1](#b-1訂正合宿は-cross-service-ではない--rucq-内部完結) の訂正参照）。

### Jomon の実ドメイン

| 概念 | 内容 |
|------|------|
| **Application（申請）** | `title`・`content`・`targets[]`(振込先＋金額)・`tags[]`・`partition`(予算区分)・`created_by` |
| **Status ライフサイクル** | `pending_review → change_requested → approved → payment_finished`（or `rejected`）。状態変更には**コメント必須** |
| **ApplicationTarget（振込対象）** | `amount`・`target`(受取人)・`paid_at`。支払い完了で自動的に `payment_finished` へ |
| **Partition / PartitionGroup** | 階層的な**予算区分**。`budget`(予算額)を持つ＝部費の予算枠管理 |
| **AccountManager（会計担当）** | `account_manager` フラグ。承認/却下・予算/タグ/ユーザー管理 |
| **Comment / Tag / File** | 申請へのコメント、分類タグ、領収書ファイル |
| **Auth** | OAuth **PKCE**（＝traQ、[11 §3](./11-trap-ecosystem.md) を裏づけ） |

### 重要な発見：Jomon は「単一申請のライフサイクル」

ユーザーの当初構想は「**事前申請 → 連鎖が止まる → 後日 精算申請**」という**二段階**だった。
しかし現状の Jomon は **1 つの Application が `pending_review→approved→payment_finished` と進む
単段階**で、「事前申請」と「精算」が別エンティティとして分かれているわけではない。
→ この差分の扱いは [08 との対応](#a-0-注-08-との対応) と [10](./10-open-questions.md) の論点（後述 J1）。

### ツール化の例

| ツール例 | 副作用クラス |
|---------|-------------|
| `jomon.applications.list` / `jomon.application.get` | read_only |
| `jomon.partition.get`（予算残の照会） | read_only |
| `jomon.application.create`（申請作成） | reversible_write |
| `jomon.application.status.update`（承認/却下=会計担当のみ） | reversible_write（承認ゲートで保護） |
| `jomon.file.upload`（領収書） | reversible_write |
| （振込実行＝`paid_at` 確定 → payment_finished） | irreversible_or_monetary |

<a id="a-0-注-08-との対応"></a>
### 注：[08 交通費精算] との対応

[08](./08-reference-workflows.md) の `pre_application.submit` / `payment.execute` は**抽象モデル**。
現実の Jomon にマップするなら：

- 「事前申請」「精算申請」を **2 つの Application（`supersedes`/`relates_to` でリンク）**として表すか、
- あるいは **1 つの Application のライフサイクル**に寄せるか、

の設計判断が必要（→ J1）。**部費に紐づく経費**である限り Jomon が舞台。部費外（合宿参加費等）は
そもそも Jomon の対象外であり、別の扱い（rucQ 等）になる。

## A-1. knoQ（部屋・イベント予約）

| ツール例 | 副作用クラス |
|---------|-------------|
| `knoq.room.availability`（空き照会） | read_only |
| `knoq.event.create` / `knoq.room.reserve` | reversible_write |
| `knoq.event.cancel` | reversible_write |

- **単発**：「来週水曜の進捗部屋を取りたい」→ 空き照会 → 予約 → traQ で関係者に通知。
- **定期**（[06](./06-recurring-tasks.md)）：「毎週○曜の定例部屋を自動予約」。区間ではなく
  `point_in_time` の繰り返し。重複予約を冪等キーで防ぐ。
- **承認**：大規模イベントや長時間占有は責任者の承認ゲート（[07](./07-human-in-the-loop.md)）。

## A-2. booQ（備品・書籍の貸出）

| ツール例 | 副作用クラス |
|---------|-------------|
| `booq.item.search`（在庫・所在照会） | read_only |
| `booq.loan.request`（貸出申請） | reversible_write |
| `booq.loan.return`（返却記録） | reversible_write |
| `booq.item.register`（資産登録） | reversible_write |

- **単発**：「撮影用カメラを借りたい」→ 在庫照会 → 貸出申請 → 返却期限を記録。
- **定期/期限**：返却期限が近づいたら traQ でリマインド（期限シグナル、[06 §8](./06-recurring-tasks.md)）。
- **承認**：高額機材は承認ゲート。
- **連携の芽**：購入した機材を `booq.item.register` で資産化（→ B-3 と接続）。

## A-3. anke-to（アンケート・投票）

| ツール例 | 副作用クラス |
|---------|-------------|
| `anketo.survey.create` | reversible_write |
| `anketo.result.fetch`（集計取得） | read_only |

- **単発**：「新歓の日程調整アンケートを作りたい」→ 作成 → 集計 →（結果を次の行動の入力に）。
- **意思決定トリガ**：投票結果が「最多日に knoQ 予約」などの後続を駆動（→ B-2）。
- **承認の代替/補完**：合議的な可否を投票で取り、ApprovalRecord に反映する設計余地。

## A-4. NeoShowcase（デプロイ／PaaS）

| ツール例 | 副作用クラス |
|---------|-------------|
| `ns.app.status`（ビルド/稼働状態照会） | read_only |
| `ns.app.redeploy` / `ns.app.restart` | irreversible_or_monetary（本番影響）|

- **単発**：「このアプリを再デプロイしたい」→ 状態照会 → 確認 → 再デプロイ。
- **承認**：本番影響のある操作は確認＋承認を必須（[07](./07-human-in-the-loop.md)）。
- **監視連動**：障害検知をシグナルに、エージェントが一次対応タスクを起票（後述 C）。

## A-5. rucQ（合宿運営）

> ⚠️ **訂正メモ**：本ドキュメント初版では rucQ を「anke-to で日程調整 → knoQ で部屋予約 →
> Jomon で集金…を束ねるハブ」と書いたが、**誤り**。実物（`openapi.yaml` ＋ `model/*.go`）を
> 確認すると、rucQ は合宿運営の大半を**自前で内包する完成度の高い専用システム**だった。
> 以下は実機能に基づく正確な記述。

### rucQ が自前で持つドメイン（実機能）

| 概念 | 内容 |
|------|------|
| **Camp** | 合宿本体。`name`・`guidebook`(Markdown のしおり)・開催期間・**ドラフト/登録受付中/支払い受付中フラグ**（＝ライフサイクル） |
| **Registration / Participant** | 参加登録・キャンセル、参加者一覧、スタッフ(`isStaff`)判定 |
| **QuestionGroup / Question / Option / Answer** | **締切(`due`)つきの事前質問**。型は free_text / free_number / single / multiple。`isPublic`/`isOpen`/`isRequired`。回答は本人提出＋管理者代理入力も可 |
| **Payment** | 参加者ごとの `amount`(請求額) と `amountPaid`(入金額)＝**集金の進捗管理** |
| **RoomGroup / Room / RoomStatus** | **部屋割り**。部屋の在室ステータスと履歴(status-logs) |
| **Event** | 合宿中のスケジュール。`duration`(時間幅) / `official`(公式) / `moment`(時点) の3型、場所・主催者付き |
| **RollCall / Reaction** | **点呼**。`subjects`(対象)・`options`(選択肢)を持ち、参加者が reaction で応答。**リアルタイムstream**あり |
| **Activity** | 部屋作成・支払い変更・点呼・質問公開などの活動フィード |
| その他 | 画像(しおり用)、管理者からの traQ DM 送信 |

### つまり：rucQ は「合宿版の本システム」に近い

rucQ は既に「ライフサイクルを持つ案件（Camp）＋締切つき入力収集（Question）＋集金（Payment）＋
割り当て（Room）＋点呼（RollCall）」を備える。本システムの抽象（タスク・状態・期限・集約）と
**構造が酷似**しており、rucQ を作り直す価値は薄い。

### では本システム（エージェント層）が rucQ に足せる価値は何か

rucQ が**データと操作**を持つのに対し、本システムは**能動的な進行と連鎖**を足す：

1. **締切の能動的な催促**（[06](./06-recurring-tasks.md), [12 C-1](#c-1-期限締切リマインドの横断)）：
   Question の `due` 前に未回答者へ traQ 通知、Payment の `amountPaid < amount` の未払い者へ催促。
   rucQ は締切と入金状況を**持つ**が、追いかけは人手。ここをエージェントが担う。
2. **点呼の異常検知**：RollCall の `subjects` と reaction を突き合わせ、未応答者を traQ で呼び出す。
3. **rucQ は合宿の収支を自己完結する（Jomon とは無関係）**：rucQ の Payment は
   **参加者からの集金**で、合宿の支出もその集めた参加費の中で扱われる。会計システム
   **Jomon は「部費（部の公式予算）」専用**であり、**合宿は Jomon を通らない**。
   したがって rucQ ⇄ Jomon の連携は基本的に発生しない。本システムが rucQ に足す価値は
   §1・§2・§4 の**内部の能動的進行（催促・点呼・ライフサイクル遷移）に絞られる**。
4. **ライフサイクルの自動遷移**：ドラフト→登録受付→支払い受付→締め、の節目に承認/通知を挟む。

ツール化の例：

| ツール例 | 副作用クラス |
|---------|-------------|
| `rucq.camp.get` / `rucq.participants.list` / `rucq.payments.list` | read_only |
| `rucq.answers.get`（未回答者の割り出し） | read_only |
| `rucq.user.message`（管理者 DM 送信） | irreversible_or_monetary（対人通知） |
| `rucq.payment.update`（入金記録の更新） | reversible_write |

## A-6. trap-collection（ゲーム管理・配布）

| ツール例 | 副作用クラス |
|---------|-------------|
| `tc.work.register`（作品登録） | reversible_write |
| `tc.work.publish`（公開） | irreversible_or_monetary（対外公開） |

- 「作品を部の展示に登録したい」→ 登録 → 公開は承認ゲート（対外公開＝不可逆）。

## A-7. traPortfolio / traPortal（参照・在籍・ロール）

- **traPortfolio**：活動・実績の参照（read_only）。タスクの文脈補強に使う。
- **traPortal**：在籍・ロールの参照は[05](./05-authz-identity.md) の認可判定の入力。
  入部処理・ロール付与は運用タスク（→ C-2 オンボーディング）。

## A-8. Qall（通話）

- **HITL のエスカレーション先**：テキスト承認が滞る／重要案件は「Qall で相談」を促す。
  承認ゲートのタイムアウト時の代替経路（[07 §4](./07-human-in-the-loop.md)）。

---

# Part B. サービス間連携（複合ワークフロー）— 本システムの核心

既存サービスは個々に完結している。**サービスをまたぐエンドツーエンドの仕事**を、一つの
タスクの連鎖として束ねるのが、このオーケストレーション層の最大の価値。

## B-1.（訂正）合宿は cross-service ではない — rucQ 内部完結

> ⚠️ **二重の訂正**：初版は「anke-to×knoQ×Jomon で合宿を組み立てる」と書き（誤り）、
> 次に「rucQ収入 ⇄ Jomon支出 の合宿収支」と書いた（**これも誤り**）。実際には——
> - rucQ は合宿運営を**自己完結**する（[A-5](#a-5-rucq合宿運営)）。
> - **Jomon は「部費（部の公式予算）」専用**で、**合宿は Jomon を通らない**（合宿は参加費で回す）。
>
> よって **合宿に cross-service の連鎖は（ほぼ）存在しない**。本システムが合宿に足せるのは
> [A-5](#a-5-rucq合宿運営) の**rucQ 内部に対する能動的進行**（締切催促・未払い催促・点呼の
> 異常検知・ライフサイクル節目の承認/通知）であり、複数サービスを束ねる話ではない。

これは設計にとって重要な一般教訓を与える：

> **教訓**：traP の各サービスは「自分のドメイン内で厚い」。安易に「サービスを束ねて○○を回す」と
> 構想すると、実際には単一サービスが既に内製している、という取り違えが起きる（合宿＝rucQ が典型）。
> 本システムの価値は二種類に整理される：
> 1. **厚いサービスの内部に対する能動的進行**（催促・点呼・ライフサイクル）＝ rucQ 型。
> 2. **本当にサービスをまたぐ薄い隙間**を埋める連鎖（次の B-2・B-3）＝ こちらは要・実機能検証。

## B-2. イベント運営の連鎖（anke-to → knoQ → traQ → Jomon）

```mermaid
graph LR
    A[日程調整 anke-to] --> B[最多日で knoQ 予約]
    B --> C[traQ で告知]
    C --> D[当日 booQ で物品手配]
    D --> E[後日 Jomon で精算]
```

- anke-to の集計結果が knoQ 予約の入力になる（**サービスの出力が次の入力**）。
- 「イベント → 物品 → 経費」が一本の連鎖で追跡可能（従来は人手で分断）。
- ⚠️ 前提：末尾の Jomon 精算は **部費で賄うイベントの場合**に限る（Jomon＝部費専用）。
  部費外のイベントなら経費は Jomon を通らない。anke-to/knoQ/booQ の機能は未検証（要精読）。

## B-3. 物品購入のライフサイクル（Jomon × booQ）

```mermaid
graph LR
    Want[機材を買いたい] --> Pre[Jomon: 事前申請（承認）]
    Pre --> Buy[購入]
    Buy --> Asset[booQ: 資産登録]
    Buy --> Settle[Jomon: 精算（領収書）]
    Asset --> Lend[以後 booQ で貸出運用]
```

- 「**部費**（Jomon）」と「資産（booQ）」を連結。購入した物が貸出可能資産として登録されるまでを
  一連で扱う（[11 §5](./11-trap-ecosystem.md) の Jomon 連携と [A-2](#a-2-booq備品書籍の貸出) が合流）。
- ⚠️ 前提：**部費で購入する機材**であること（Jomon＝部費専用）。これは Jomon の本来用途に合致する、
  比較的確度の高い cross-service 例。ただし booQ 側の登録 API は未検証（要精読）。
- 注：Jomon の単一申請ライフサイクル（[A-0](#a-0-jomon部費会計-実機能を検証済み)）に照らすと、
  「事前申請（承認）」と「精算（領収書）」を 1 申請で表すか 2 申請で表すかは J1 の論点。

## B-4. 作品公開フロー（NeoShowcase × trap-collection × anke-to）

- NeoShowcase でデプロイ → trap-collection で作品登録・公開（承認）→ anke-to で部内評価投票。
- ハッカソン後の「成果を公開して評価を集める」を一本化。

---

# Part C. まだ拾えていない横断ユースケース

個別サービスに紐づかない、**運用・組織レベル**の用途。ここに定期タスク・承認・委譲の価値が効く。

## C-1. 期限・締切リマインドの横断

booQ 返却期限・Jomon 精算期限・イベント申込締切・合宿支払い…を**横断的に**監視し、
期限シグナル（[06 §8](./06-recurring-tasks.md)）で traQ 通知・エスカレーション。
「締切管理」を個別サービス任せにしない。

## C-2. オンボーディング（入部時の一括処理）

「新入部員を迎える」一つのタスクが、traPortal 登録 → traQ 招待 → ロール付与 →
初期チャンネル案内…を順に実行。**人間の確認を要所に挟む**自動化。

## C-3. 役職・権限の引き継ぎ（認可平面の運用タスク）

年度替わりの役職交代を、権限の**委譲・棚卸し**として扱う（[05 §3](./05-authz-identity.md)）。
「会計担当を交代する」＝ Jomon 承認権限の付け替え＋旧権限の失効を、監査可能な形で実行。

## C-4. 棚卸し・監査

年度末の物品棚卸し（booQ）、会計監査（Jomon）を定期タスク化。台帳（[02 §8](./02-domain-model.md)）が
監査証跡を提供。

## C-5. 「相談 → 意思決定 → 実行」の汎用パターン

anke-to 投票や Qall 相談で合意を取り、その結果を実行に移す——という、ドメイン非依存の
意思決定パターン。経理にも、イベントにも、組織運営にも適用できる本システムの素地。

## C-6. 外部対応タスク

スポンサー連絡、大学事務への申請、他団体との調整など、**対外（不可逆送信）**を伴う連絡を
[08 §3](./08-reference-workflows.md) の外部連絡フローで扱う。

---

# Part D. 連携を支える設計上の含意

横断連携を成立させるために、既存ドキュメントへ反映すべき点。

1. **クロスサービス・タスクリンク**（[02 §4](./02-domain-model.md), [11 §9 N4](./11-trap-ecosystem.md)）：
   本システムの「タスク」を、Jomon の申請・knoQ の予約・booQ の貸出など**既存サービスの
   レコードへリンク**できる必要がある。`external_ref`（サービス名＋ID）を持つリンク型を検討。
2. **統一ツールカタログ**（[04 §5](./04-tool-integration.md)）：全サービス API を副作用クラス付きで
   登録。下表が初期分類。
3. **traQ スコープでの横断アクセス**（[05](./05-authz-identity.md), [11 §3](./11-trap-ecosystem.md)）：
   各サービスは traQ OAuth2 で認証済み。エージェントはユーザー委譲の traQ トークンで横断する。
4. **集約は サブワークフロー**（[03 §6](./03-workflow-engine.md)）：合宿集金・月次精算などの
   ファンアウト→ファンインで表現。

## 全サービス 副作用クラス分類（初期案）

| サービス | read_only | reversible_write | irreversible_or_monetary |
|---------|-----------|------------------|--------------------------|
| knoQ | 空き照会 | 予約・取消 | （長期占有は承認で扱う） |
| booQ | 在庫照会 | 貸出・返却・登録 | 高額機材の貸出（承認） |
| anke-to | 集計取得 | アンケート作成 | — |
| Jomon | 残高・申請照会 | 申請下書き | **支払い・精算** |
| NeoShowcase | 状態照会 | 設定変更 | **本番デプロイ・再起動** |
| trap-collection | 作品照会 | 作品登録 | **公開（対外）** |
| traQ | メッセージ取得 | 下書き | **対外送信・送金性メッセージ** |

---

## この探索から生まれた未決事項（[10](./10-open-questions.md) に追記予定）

- **U1**：複合ワークフロー（合宿・イベント）を MVP に含めるか、単一サービス縦串から始めるか。
- **U2**：`external_ref`（外部レコードへのリンク）の表現と、重複・同期の扱い（N4 の具体化）。
- **U3**：anke-to 投票を「承認」とみなせる条件（合議の定足数・期限）。
- **U4**：横断リマインド（C-1）を独立機能にするか、各タスクの期限から自動生成するか。
- **U5**：オンボーディング/役職引き継ぎ（C-2/C-3）を業務テンプレートとして標準提供するか。

## 参考リンク

- SysAd 班（サービス運用）: https://trap.jp/sysad/
- 新入部員向けサービス全解説: https://trap.jp/post/2154/
- knoQ: https://github.com/traPtitech/knoQ ／ booQ: https://github.com/traPtitech/booQ
- anke-to: https://github.com/traPtitech/anke-to ／ NeoShowcase: https://github.com/traPtitech/NeoShowcase
- traPortfolio: https://portfolio.trap.jp/ ／ trap-collection: https://github.com/traPtitech/trap-collection-server
