# crispy-waffle

グラフベースの**タスク／ワークフロー処理アプリ**。タスクを独立した部品ではなく、
**有向グラフの頂点**として扱う。ある頂点から次の頂点へ進むには、その頂点に定義された
「何かしらのタスク」が満たされる必要がある。大きなタスクは小さなステップに分割され、
ステップが一つずつ進むことで全体が進展する。

このリポジトリの `docs/` にある設計（work / execution / authorization の三平面）を、
**Go（バックエンド）＋ React/Vite/TypeScript（フロントエンド）** で実装したもの。

> 重要な思想: タスクを進めるのは LLM エージェントではなく**ただのシステム**。
> エージェント（LLM やプログラム）は、あるステップを満たすために起動される**一実行主体**に
> すぎない。人・機械・プログラムはすべて同列で、すべてが一様に「タスク（ステップ）」として
> 描画される。

## 何ができるか

リファレンス業務として**交通費の精算**（`expense.transport`）を一本通せる:

1. タスク作成 → 2. システムが運賃を照会（`fare.lookup`）→ 3. 実行者へ確認（Yes/No, *Elicitation*）→
4. 事前申請を提出（`pre_application.submit`）→ 5. **会計担当の承認ゲート**（連鎖が一旦停止＝
durable wait）→ 6. 後日「精算する」→ 7. 冪等な支払い実行（`payment.execute`）→ 8. 完了。

中核概念:

- **イベント台帳が正本** — `現在状態 = Fold(events)`（イベントソーシング）。状態は黙って変わらない。
- **durable pause/resume** — 人間ゲートで実行は眠り、承認/トリガのシグナルで再開する。
- **承認は第一級** — 会話の発言ではなく構造化された `ApprovalRecord`。承認面は最小文脈のみ。
- **認可ゲート** — ツール実行は「ステップの束縛 ∩ アクターのケイパビリティ」の範囲内のみ。
- **型つきタスクグラフ** — `subtask_of` / `blocks` / `approval_for` 等の有向辺。
- **冪等な金銭操作** — 台帳の冪等キーで二重送金を防止。

## アーキテクチャ

```
cmd/server         配線（DB→マイグレーション→シード→HTTP）
internal/
  domain           WORK平面: Task / TaskLink / ApprovalRecord / Actor / Capability（純粋型）
  workflow         EXECUTION平面: 定義・イベント・Fold・エンジン（durable advance）・ガード・シグナル
  authz            AUTHORIZATION平面: binding ∩ capability の判定
  exec             実行主体（差し込み口）。決定論モックのツール実装（オフラインで動く）
  tools            ツールカタログ（副作用クラス）
  store            SQLite（modernc, CGO不要）追記専用イベント台帳＋射影
  service          ドメイン・エンジン・ストアのオーケストレーション
  api              net/http REST（X-Actor-Id スタブ認証）
  seed             expense.transport@1 リファレンス定義＋デモタスク
frontend           React + Vite + TS（ダッシュボード / 詳細 / 承認インボックス）
```

技術選定: 永続化は SQLite ファイル（`modernc.org/sqlite`、CGO 不要）。`agent_action` は
LLM 必須ではなく、`Executor` インターフェースの一実装。実 LLM は同インターフェースで差し替え可能。

## 動かし方

前提: Go 1.24+, Node 22+。

```bash
# 依存解決＋フロントのビルド（任意。dist があるとバックエンドが SPA も配信する）
make tidy
make install-frontend

# 開発モード（バックエンド :8080 とフロント :5173 を同時起動。Vite が /api をプロキシ）
make dev
# → ブラウザで http://localhost:5173

# もしくはバックエンドのみ（ビルド済み frontend/dist があれば http://localhost:8080 で完結）
make build         # bin/server とフロント bundle を生成
make run-backend
```

DB は `./data/app.db` に作られる。作り直すには `make reset-db`。

## エンドツーエンド検証

1. `make run-backend` → ログに `migrated` / `seeded expense.transport@1` / `listening :8080`。
2. `curl localhost:8080/api/tasks` でデモタスク、`/api/tools` で3ツール。
3. フロントを開く（`make dev` なら :5173）。ダッシュボードにデモタスクが出る。
4. タスクを開く → 「はい（事前申請する）」→ タイムラインに `tool.invoked fare.lookup` と金額、状態が
   **会計承認待ち**に。
5. 右上の担当者を**会計担当**に切替 → 承認インボックスに1件 → **承認**。状態が **AwaitingSettlement** に。
6. 担当者を**田中**に戻す → タスク詳細で **精算する** → `payment.execute`（冪等）→ **Settled**。
   もう一度「精算する」を押しても二重送金しない（実行は完了済み）。

テスト:

```bash
make test   # Fold 再現性 / ガード評価 / 精算フロー（成功・却下・確認ループ・冪等）
```

## ツールの追加（コードで増やす）

ツールは「ステップが叩ける能力」で、**コードで1ファイル書いて1行登録**すれば増えます。
データ駆動の UI ビルダーは持たず、汎用の `Tool` インターフェースに集約しています。
登録した Spec が、カタログ API・フロービルダーのツール選択・認可・実行のすべての源になります。

1. `internal/tools/builtin/` に `tools.Tool` を実装した型を作る:

```go
type ReceiptCheck struct{}

func (ReceiptCheck) Spec() tools.Spec {
    return tools.Spec{
        Key: "receipt.check", DisplayName: "領収書チェック",
        SideEffectClass: tools.ReadOnly,
        Inputs:  []tools.Field{{Name: "receipt_attached", Type: "boolean"}},
        Outputs: []tools.Field{{Name: "receipt_ok", Type: "boolean"}},
    }
}

func (ReceiptCheck) Execute(ctx context.Context, in tools.Input) (map[string]any, error) {
    ok, _ := in.Context["receipt_attached"].(bool)
    return map[string]any{"receipt_ok": ok}, nil
}
```

2. `internal/tools/builtin/builtin.go` の `Register` に1行追加: `r.Register(ReceiptCheck{})`

これだけで、`GET /api/tools` に出てフロービルダーで束縛でき、実行されます。実体（HTTP 呼び出し・
traQ 通知・DB 参照など）は `Execute` の中に書きます。`SideEffectClass` は認可と、フロー設計時の
注意喚起（不可逆ツールは確認/承認ゲートの後ろに置く）に使われます。`amount` のようなスコープ次元は
`ScopeDimensions` で宣言すると、認可ゲートがアクターのケイパビリティ上限（例: `amount_cap`）で
束縛します。組み込み例として `fare.lookup` / `pre_application.submit` / `payment.execute` /
`notify.send` を同梱しています。

> 既定エージェント `acc-bot` には起動時に全登録ツールのケイパビリティが自動付与されるため、
> コードで足したツールはそのまま使えます。

## 設計ドキュメント

概念の詳細は [`docs/`](./docs)（ビジョン / ドメインモデル / 実行モデル / HITL / リファレンス
ワークフロー 等）を参照。本実装はその語彙（リンク型・ステップ種別・ステータス・副作用クラス）を
そのまま用いている。
