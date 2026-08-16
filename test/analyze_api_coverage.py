#!/usr/bin/env python3
"""
Gitee API v5 与 gitee-cli 功能覆盖对比分析工具
"""
import json
import urllib.request
import re
from collections import defaultdict

def fetch_swagger():
    url = "https://gitee.com/api/v5/swagger_doc.json"
    req = urllib.request.Request(url, headers={"User-Agent": "gitee-cli-qa/1.0"})
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.loads(resp.read().decode("utf-8"))

def extract_endpoints(swagger):
    """按 tag 提取所有端点"""
    by_tag = defaultdict(list)
    paths = swagger.get("paths", {})
    
    for path, methods in paths.items():
        for method, spec in methods.items():
            if method.lower() not in ("get", "post", "put", "patch", "delete"):
                continue
            summary = spec.get("summary", "")
            tags = spec.get("tags", ["Unknown"])
            tag = tags[0] if tags else "Unknown"
            op_id = spec.get("operationId", "")
            by_tag[tag].append({
                "path": path,
                "method": method.upper(),
                "summary": summary,
                "op_id": op_id,
            })
    return dict(by_tag)

def main():
    print("Fetching Gitee API v5 swagger_doc.json...")
    swagger = fetch_swagger()
    print(f"API Version: {swagger.get('info', {}).get('version', 'unknown')}")
    
    endpoints = extract_endpoints(swagger)
    
    # 项目已实现的功能映射（基于已有命令和 internal/api/endpoints.go）
    implemented = {
        "Activity": [
            "notifications/count", "notifications/threads",
            # watch/star 相关在 user/repo 下
        ],
        "Repositories": [
            "/repos/{owner}/{repo}/branches", "/repos/{owner}/{repo}/branches/{branch}",
            "/repos/{owner}/{repo}/branches/{branch}/protection",
            "/repos/{owner}/{repo}/tags",
            "/repos/{owner}/{repo}/comments", "/repos/{owner}/{repo}/commits/{ref}/comments",
            "/repos/{owner}/{repo}/comments/{id}", "/repos/{owner}/{repo}/commits/{sha}/comments",
            "/repos/{owner}/{repo}/commits", "/repos/{owner}/{repo}/commits/{sha}",
            "/repos/{owner}/{repo}/compare/{base}...{head}",
            "/repos/{owner}/{repo}/keys", "/repos/{owner}/{repo}/keys/available",
            "/repos/{owner}/{repo}/keys/enable/{id}", "/repos/{owner}/{repo}/keys/{id}",
            "/repos/{owner}/{repo}/readme",
            "/repos/{owner}/{repo}/contents(/{path})", "/repos/{owner}/{repo}/contents/{path}",
            "/repos/{owner}/{repo}/blame/{path}", "/repos/{owner}/{repo}/raw/{path}",
            "/repos/{owner}/{repo}/zipball", "/repos/{owner}/{repo}/tarball",
            "/repos/{owner}/{repo}",  # get/edit/delete/archive/rename
            "/repos/{owner}/{repo}/forks",
            "/repos/{owner}/{repo}/pull",  # sync
            "/repos/{owner}/{repo}/releases", "/repos/{owner}/{repo}/releases/tags/{tag_name}",
            "/repos/{owner}/{repo}/releases/{id}/assets",
            "/repos/{owner}/{repo}/collaborators",
            "/user/repos", "/users/{owner}/repos",
            "/orgs/{org}/repos",
            "/user/starred/{owner}/{repo}", "/user/subscriptions/{owner}/{repo}",
            "/user/starred",
            "/repos/{owner}/{repo}/branches/{wildcard}/setting",
            "/repos/{owner}/{repo}/branches/setting/new",
        ],
        "Issues": [
            "/repos/{owner}/{repo}/issues", "/repos/{owner}/{repo}/issues/{number}",
            "/repos/{owner}/{repo}/issues/{number}/comments",
            "/repos/{owner}/{repo}/issues/comments", "/repos/{owner}/{repo}/issues/comments/{id}",
            "/user/issues", "/orgs/{org}/issues",
        ],
        "Labels": [
            "/repos/{owner}/{repo}/labels",
        ],
        "Milestones": [
            "/repos/{owner}/{repo}/milestones", "/repos/{owner}/{repo}/milestones/{number}",
        ],
        "Pull Requests": [
            "/repos/{owner}/{repo}/pulls", "/repos/{owner}/{repo}/pulls/{number}",
            "/repos/{owner}/{repo}/pulls/{number}/merge",
            "/repos/{owner}/{repo}/pulls/{number}/review",
            "/repos/{owner}/{repo}/pulls/{number}/comments",
        ],
        "Gists": [
            "/gists", "/gists/{gist_id}", "/gists/{gist_id}/star",
            "/gists/{gist_id}/comments",
        ],
        "Organizations": [
            "/user/orgs", "/orgs/{org}", "/orgs/{org}/members", "/orgs/{org}/repos",
            "/orgs/{org}/issues",
        ],
        "Search": [
            "/search/repositories", "/search/issues", "/search/users",
            "/search/code", "/search/commits",
        ],
        "Users": [
            "/user/keys", "/user/keys/{id}",
            "/user", "/users/{username}",
            "/users/{username}/followers", "/users/{username}/following",
            "/user/following/{username}",
        ],
        "Webhooks": [
            "/repos/{owner}/{repo}/hooks", "/repos/{owner}/{repo}/hooks/{id}",
            "/repos/{owner}/{repo}/hooks/{id}/tests",
        ],
        "Checks": [
            "/repos/{owner}/{repo}/check-runs",
            "/repos/{owner}/{repo}/check-runs/{check_run_id}",
            "/repos/{owner}/{repo}/check-runs/{check_run_id}/annotations",
            "/repos/{owner}/{repo}/commits/{ref}/check-runs",
        ],
        "Emails": [],
        "Enterprises": [],
        "Git Data": [],
        "Miscellaneous": [],
    }
    
    # CLI 命令到 tag 的映射
    cli_commands = {
        "auth": ["Users"],
        "alias": [],
        "api": [],  # 通用 raw API
        "completion": [],
        "config": [],
        "extension": [],
        "gist": ["Gists"],
        "issue": ["Issues"],
        "label": ["Labels"],
        "milestone": ["Milestones"],
        "notification": ["Activity"],
        "org": ["Organizations", "Enterprises"],
        "pr": ["Pull Requests"],
        "release": ["Repositories"],
        "repo": ["Repositories"],
        "search": ["Search"],
        "ssh-key": ["Users"],
        "status": [],
        "user": ["Users"],
        "webhook": ["Webhooks"],
    }
    
    print("\n" + "="*70)
    print("Gitee API v5 端点统计")
    print("="*70)
    
    total_endpoints = 0
    for tag, eps in sorted(endpoints.items()):
        count = len(eps)
        total_endpoints += count
        print(f"\n[{tag}] — {count} 个端点")
        for ep in eps[:5]:  # 只显示前5个
            print(f"  {ep['method']:6} {ep['path']:<50} {ep['summary']}")
        if len(eps) > 5:
            print(f"  ... 还有 {len(eps)-5} 个端点")
    
    print(f"\n总计: {total_endpoints} 个端点")
    
    print("\n" + "="*70)
    print("功能覆盖分析")
    print("="*70)
    
    for tag, eps in sorted(endpoints.items()):
        cli_modules = [cmd for cmd, tags in cli_commands.items() if tag in tags]
        print(f"\n[{tag}]")
        print(f"  API 端点数量: {len(eps)}")
        print(f"  CLI 模块: {', '.join(cli_modules) if cli_modules else '无'}")
        
        # 简单匹配看看有多少端点被覆盖
        covered = 0
        for ep in eps:
            path = ep["path"]
            # 检查是否已知的端点
            for known in implemented.get(tag, []):
                # 简单通配符匹配
                pattern = known.replace("{", "[^/]+").replace("}", "")
                if re.search(pattern.replace("...", "\\.\\.\\."), path):
                    covered += 1
                    break
        print(f"  估算覆盖: {covered}/{len(eps)} ({covered*100//len(eps) if len(eps) > 0 else 0}%)")

if __name__ == "__main__":
    main()
