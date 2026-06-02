# crispy-waffle 設計ドキュメント

> **English summary**: This directory holds the design and research notes for an
> *LLM-agent-driven task & workflow system* — an alternative to chat-first AI
> products. Work lives in durable, linkable **task** entities (like Git issues);
> **agents** execute steps by calling external **tools** under per-actor
> **capability** scoping; **approval gates** pause and resume task chains; and
> **recurring tasks** fire on schedules with interval ("since last run") semantics.
> All design is intentionally technology-agnostic. Documents are written in Japanese.

このディレクトリは、本プロダクトが「何を・なぜ・どう作るか」を検討するための
設計・調査ドキュメント群です。実装に入る前に、**データ構造**・**外部ツール統合**・
**権限／認証認可**・**ワークフロー実行**・**具体的な業務フロー**を、特定の技術スタックに
依存しない形で言語化することを目的としています。

> 🎯 **想定ユーザー**：本プロダクトは **traP（東京科学大学デジタル創作同好会、約250名）**
> とその OSS 群（traQ・Jomon・knoQ など）の利用者を対象とします。設計は技術非依存で進めつつ、
> 実環境への接地は [11-trap-ecosystem.md](./11-trap-ecosystem.md) に集約しています。
> **人間の認証は traQ の OAuth2 認可サーバーに委ね、対話・承認は traQ 上で行い、Jomon 等の
> 既存サービスはツールとしてエージェントが叩く**——という立ち位置です。

## このプロダクトは何か（1段落で）

現在の LLM 活用の主流は「チャット」です。しかしチャットは、**作業の途中で第三者の承認を
得る**、**長期間にわたって状態を保持する**、**定期的に同じ処理を繰り返す**、**実行者ごとに
できることを変える**といった「組織の業務」を素直に表現できません。本プロダクトは、これらを
**永続的なタスク・エンティティ**と**それを駆動するワークフロー**として表現し、各ステップを
**LLM エージェントがツールを叩いて実行する**仕組みです。チャットは数ある入力手段の一つに
過ぎず、本体は「タスクの台帳」と「権限つきの実行基盤」です。

## ドキュメント一覧

| # | ファイル | 内容 |
|---|---------|------|
| 00 | [00-vision.md](./00-vision.md) | ビジョン・課題設定（なぜチャットだけでは不十分か） |
| 01 | [01-glossary.md](./01-glossary.md) | 用語集・主要概念の定義 |
| 02 | [02-domain-model.md](./02-domain-model.md) | ドメインモデル／データ構造（タスク・リンク・ワークフロー・承認） |
| 03 | [03-workflow-engine.md](./03-workflow-engine.md) | ワークフロー実行モデル（状態機械・中断と再開・イベントソーシング） |
| 04 | [04-tool-integration.md](./04-tool-integration.md) | 外部ツール統合（エージェントによるツール利用・ツールカタログ） |
| 05 | [05-authz-identity.md](./05-authz-identity.md) | 認証・認可・権限管理（OIDC / OAuth / 委譲 / ReBAC / ケイパビリティ） |
| 06 | [06-recurring-tasks.md](./06-recurring-tasks.md) | 定期タスク（スケジューリングと区間処理の意味論） |
| 07 | [07-human-in-the-loop.md](./07-human-in-the-loop.md) | 人間の承認・介入（承認ゲート・確認 UX） |
| 08 | [08-reference-workflows.md](./08-reference-workflows.md) | 具体ワークフロー（経費精算：事前申請→会計承認→精算） |
| 09 | [09-research-notes.md](./09-research-notes.md) | 調査メモ（外部の一次情報・標準・先行事例のリンクと要約） |
| 10 | [10-open-questions.md](./10-open-questions.md) | 未決事項・ユーザーへの確認リスト |
| 11 | [11-trap-ecosystem.md](./11-trap-ecosystem.md) | **traP エコシステムへの接地**（想定ユーザー・traQ/Jomon 連携） |

## 設計の三層モデル（読む前の地図）

本ドキュメント群を貫く中心的な考え方は、関心を 3 つの平面に分離することです。

```
┌─────────────────────────────────────────────────────────┐
│  作業平面 (Work Plane)                                    │
│   「仕事はどこに在るか」                                  │
│   タスク／サブタスク／リンク／承認レコード／台帳          │
│   → 永続的・人間に可視・監査可能                          │
├─────────────────────────────────────────────────────────┤
│  実行平面 (Execution Plane)                               │
│   「仕事はどう進むか」                                    │
│   ワークフロー定義／実行インスタンス／ステップ／状態機械  │
│   → 中断と再開ができる・イベントで駆動される              │
├─────────────────────────────────────────────────────────┤
│  認可平面 (Authorization Plane)                           │
│   「誰が何をしてよいか」                                  │
│   アクター（人・エージェント）／アイデンティティ／        │
│   ケイパビリティ／ツールごとのスコープ                    │
│   → 委譲は常に縮小（attenuation）・短命・監査可能          │
└─────────────────────────────────────────────────────────┘
```

この分離により、「タスクの意味」「実行の仕組み」「権限の管理」を独立に進化させられます。

## 進め方（このリポジトリの運用方針）

1. `docs/` に検討した MD を蓄積していく（本コミットがその第一弾）。
2. 各 MD はレビューと議論のたたき台。確定ではなく**検討中**を前提とする。
3. 未決事項は [10-open-questions.md](./10-open-questions.md) に集約し、ユーザー確認の上で更新する。
4. 設計が十分に固まった段階で、初めて技術スタックとプロダクトのスコープを確定する。

> ⚠️ 注: 本ドキュメントは現時点(2026-06)での外部標準・コミュニティ動向の調査に基づきます。
> AI エージェントのアイデンティティ／認可は標準化の途上にあり、参照先は流動的です。
> 一次情報は [09-research-notes.md](./09-research-notes.md) を参照してください。
