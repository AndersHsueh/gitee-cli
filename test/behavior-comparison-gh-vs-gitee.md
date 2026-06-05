# gh vs gitee-cli 行为对比测试报告

> **生成时间**: 2026-06-05 21:34:03
> **gh 版本**: gh version 2.74.0 (2025-05-29)
> **gitee-cli 版本**: gitee version 0.4.2
> **测试范围**: 全量 (17 个一级命令 + 全部 verb)

## 1. 总览

| 指标 | 数字 |
|------|------|
| gitee 一级命令 | 17 |
| gh 一级命令 | 29 |
| --agent-help 声明 verb 数 | 37 |
| **通过** | **155** |
| **失败** | **2** |
| **跳过** | **1** |
| 总计 | 158 |

## 2. 一级命令覆盖

| gitee-cli 一级 | gh 等价 | 状态 |
|---|---|---|
| `api` | `api` | ✅ |
| `auth` | `auth` | ✅ |
| `completion` | `completion` | ✅ |
| `config` | `config` | ✅ |
| `gist` | `gist` | ✅ |
| `gpg-key` | `gpg-key` | ✅ |
| `issue` | `issue` | ✅ |
| `label` | `label` | ✅ |
| `org` | `org` | ✅ |
| `pr` | `pr` | ✅ |
| `release` | `release` | ✅ |
| `repo` | `repo` | ✅ |
| `search` | `search` | ✅ |
| `ssh-key` | `ssh-key` | ✅ |
| `status` | `status` | ✅ |
| `version` | `—` | ✅ |
| `webhook` | `(在 api 后)` | ✅ |

## 3. --agent-help 14 章节验证

| verb | 状态 | 缺失章节 |
|---|---|---|
| `repo archive` | ✅ | — |
| `repo sync` | ✅ | — |
| `repo rename` | ✅ | — |
| `issue create` | ✅ | — |
| `issue comments` | ✅ | — |
| `pr create` | ✅ | — |
| `pr comment` | ✅ | — |
| `pr checkout` | ✅ | — |
| `pr ready` | ✅ | — |
| `pr edit` | ✅ | — |
| `pr checks` | ✅ | — |
| `gist clone` | ✅ | — |
| `ssh-key list` | ✅ | — |
| `ssh-key add` | ✅ | — |
| `ssh-key delete` | ✅ | — |
| `gpg-key list` | ✅ | — |
| `gpg-key add` | ✅ | — |
| `gpg-key delete` | ✅ | — |
| `webhook list` | ✅ | — |
| `webhook create` | ✅ | — |
| `webhook view` | ❌ | 缺可选: ## Example |
| `webhook edit` | ✅ | — |
| `webhook delete` | ✅ | — |
| `webhook test` | ❌ | 缺可选: ## Example |
| `release list` | ✅ | — |
| `release view` | ✅ | — |
| `release create` | ✅ | — |
| `release edit` | ✅ | — |
| `release delete` | ✅ | — |
| `release download` | ✅ | — |
| `label list` | ✅ | — |
| `label view` | ✅ | — |
| `label create` | ✅ | — |
| `label edit` | ✅ | — |
| `label delete` | ✅ | — |
| `label clone` | ✅ | — |
| `status` | ✅ | — |

## 4. 行为差异 (非 bug)

- issue comments --agent-help 需要 positional (1), 用占位符验证
- repo list json keys 差异: gitee=[created_at default_branch description fork forks_count full_name homepage html_url id language name open_issues_count owner private pushed_at stars_count updated_at watchers_count] vs gh=[createdAt description isPrivate name]
- issue list json keys 差异: gitee=[assignee body comments created_at finished_at html_url id labels milestone number state title updated_at user] vs gh=[author createdAt number state title]
- pr list json keys 差异: gitee=[additions assignees base body changed_files comments commits created_at deletions diff_url draft head html_url id labels locked mergeable merged milestone number patch_url state title updated_at user] vs gh=[author createdAt number state title]
- 字段命名风格: gitee 用 snake_case (Gitee API 原生); gh 用 camelCase. 这是 API 风格差异, 不是 bug.
- --agent-help 是 gitee 独有; gh 无此 flag
- gitee 一级命令里把 webhook 提升为顶级 (gh 把它藏在 api 后)

## 5. 真实 Bug 候选 (如发现)

- 失败: Error: listing GPG keys: HTTP 404: resource does not exist or you lack read access
- 失败: panic: runtime error: index out of range [1] with length 1

goroutine 1 [running]:
gitee-cli/cmd/api.runAPI(0x102f115a0, {0x1400023c510, 0x1, 0x102a38854?})
	/Users/xueyuheng/powerful-app/gitee-cli/cm

## 6. 真实 API 抽查

| 命令 | 状态 | 备注 |
|---|---|---|
| `gitee repo list --limit 1 --output json` | ✅ | — |
| `gitee issue list --repo andershsueh/outsourcing-ms --limit 1 --output json` | ✅ | — |
| `gitee pr list --repo andershsueh/outsourcing-ms --limit 1 --output json` | ✅ | — |
| `gitee gist list --limit 1 --output json` | ✅ | — |
| `gitee org list --limit 1 --output json` | ✅ | — |
| `gitee release list --repo andershsueh/outsourcing-ms --limit 1 --output json` | ✅ | — |
| `gitee label list --repo andershsueh/outsourcing-ms --output json` | ✅ | — |
| `gitee search repos gitee-cli --limit 1 --output json` | ✅ | — |
| `gitee search issues bug --limit 1 --output json` | ✅ | — |
| `gitee ssh-key list --output json` | ✅ | — |
| `gitee gpg-key list --output json` | ❌ | — |
| `gitee webhook list --repo andershsueh/outsourcing-ms --output json` | ✅ | — |
| `gitee api user --output json` | ❌ | — |

## 7. 复现方式

```bash
# 1. 编译
cd /Users/xueyuheng/powerful-app/gitee-cli
go build -o gitee ./cmd/gitee

# 2. 设置 token (私密, 不入 git)
export GITEE_TEST_TOKEN=<your-token>

# 3. 跑测试
./test/comparison-test.sh

# 4. 查看报告
cat test/behavior-comparison-gh-vs-gitee.md
```

---

_本报告由 `test/comparison-test.sh` 自动生成, 不可手改._
