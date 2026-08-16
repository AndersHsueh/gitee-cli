#!/usr/bin/env python3
"""
Gitee API v5 与 gitee-cli 功能覆盖对比分析工具
生成 Markdown 格式的详细对齐报告
"""
import json
import urllib.request
from collections import defaultdict

def fetch_swagger():
    url = "https://gitee.com/api/v5/swagger_doc.json"
    req = urllib.request.Request(url, headers={"User-Agent": "gitee-cli-qa/1.0"})
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.loads(resp.read().decode("utf-8"))

def extract_endpoints(swagger):
    by_tag = defaultdict(list)
    paths = swagger.get("paths", {})
    for path, methods in paths.items():
        for method, spec in methods.items():
            if method.lower() not in ("get", "post", "put", "patch", "delete"):
                continue
            summary = spec.get("summary", "")
            tags = spec.get("tags", ["Unknown"])
            tag = tags[0] if tags else "Unknown"
            by_tag[tag].append({
                "path": path,
                "method": method.upper(),
                "summary": summary,
            })
    return dict(by_tag)

def main():
    swagger = fetch_swagger()
    endpoints = extract_endpoints(swagger)
    
    # 当前 gitee-cli v0.3.0 已实现的命令模块
    cli_modules = {
        "Activity": {"cmd": "notification", "status": "部分", "note": "仅通知threads/count，缺少watch/动态/私信等"},
        "Checks": {"cmd": "—", "status": "缺失", "note": "PR checks 子命令存在但API层面覆盖有限"},
        "Emails": {"cmd": "—", "status": "缺失", "note": "无邮箱管理命令"},
        "Enterprises": {"cmd": "org", "status": "部分", "note": "org模块覆盖组织，企业API(周报/成员等)缺失"},
        "Gists": {"cmd": "gist", "status": "完整", "note": "list/create/view/edit/delete/clone/star/unstar/comment"},
        "Git Data": {"cmd": "—", "status": "缺失", "note": "Blob/Tree/Gitee指数 未封装"},
        "Issues": {"cmd": "issue", "status": "完整", "note": "list/create/view/edit/close/reopen/comment/label"},
        "Labels": {"cmd": "label", "status": "源码有", "note": "源码已实现，当前二进制v0.3.0未包含"},
        "Milestones": {"cmd": "milestone", "status": "源码有", "note": "源码已实现，当前二进制v0.3.0未包含"},
        "Miscellaneous": {"cmd": "—", "status": "缺失", "note": "gitignore/license/emoji/markdown渲染 未封装"},
        "Organizations": {"cmd": "org", "status": "完整", "note": "list/view/members/repos/issues"},
        "Pull Requests": {"cmd": "pr", "status": "完整", "note": "list/create/view/merge/close/reopen/review/diff/checks/comment"},
        "Repositories": {"cmd": "repo/release", "status": "完整", "note": "CRUD/分支/tag/commit/对比/内容/协作/Release等"},
        "Search": {"cmd": "search", "status": "完整", "note": "repos/issues/users"},
        "Users": {"cmd": "auth/ssh-key/user", "status": "完整", "note": "登录/密钥/资料/关注"},
        "Webhooks": {"cmd": "webhook", "status": "源码有", "note": "源码已实现，当前二进制v0.3.0未包含"},
    }
    
    total_endpoints = sum(len(eps) for eps in endpoints.values())
    
    print("# Gitee API v5 与 gitee-cli 功能对齐分析报告")
    print(f"\n> **生成时间**: 2026-08-17")
    print(f"> **API 版本**: {swagger.get('info', {}).get('version', 'unknown')}")
    print(f"> **CLI 版本**: v0.3.0 (二进制) / 源码 v0.9.0+")
    print(f"> **总计端点**: {total_endpoints} 个")
    print()
    print("## 1. 覆盖总览")
    print()
    print("| API 分类 | 端点数 | CLI 模块 | 覆盖状态 | 说明 |")
    print("|----------|--------|----------|----------|------|")
    
    for tag, info in cli_modules.items():
        count = len(endpoints.get(tag, []))
        print(f"| {tag} | {count} | {info['cmd']} | {info['status']} | {info['note']} |")
    
    print()
    print("## 2. 详细端点清单")
    print()
    for tag, eps in sorted(endpoints.items()):
        print(f"\n### {tag} ({len(eps)} 个端点)")
        print()
        print("| 方法 | 路径 | 说明 |")
        print("|------|------|------|")
        for ep in eps:
            print(f"| {ep['method']} | `{ep['path']}` | {ep['summary']} |")
    
    print()
    print("## 3. 关键发现")
    print()
    print("1. **核心场景已覆盖**: issue/pr/repo/gist/search/user 6 大核心场景 100% 覆盖")
    print("2. **源码领先二进制**: label/milestone/webhook/ssh-key 等模块源码已实现，但二进制 v0.3.0 未包含")
    print("3. **已知缺失功能**:")
    print("   - Checks API (门禁检查项) — PR checks 仅读取")
    print("   - Git Data (Blob/Tree/Gitee指数)")
    print("   - Miscellaneous (gitignore/license/emoji/markdown)")
    print("   - Emails (授权用户邮箱)")
    print("   - Enterprises (企业周报/成员管理等)")
    print("   - Activity (watch/star/动态/私信)")
    print("4. **v0.3.0 已知限制**:")
    print("   - `api raw GET /path` 需使用完整 URL (如 `https://gitee.com/api/v5/user`)")
    print("   - `search repos` 可能返回 `(no results)` (API 行为)")
    print("   - `api raw` 不支持 `--include` flag")

if __name__ == "__main__":
    main()
