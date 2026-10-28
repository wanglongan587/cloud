# squads：小队管理入口

[English](README.en.md)

真实小队管理：列表、成员职责、维护者、leader 和归档历史均来自当前工作区 REST 接口。已有成员、外部新人和 Space Agent 分开添加；角色仍由租户权威决定。页面不读取演示数据，不虚构人数、在线状态或可执行能力，配置 leader 不代表可以自动执行。

| 文件 | 职责 |
| --- | --- |
| `api.ts` | 生成客户端、分页读取、版本化命令及查询失效 |
| `squads-page.tsx`、`squad-detail-page.tsx` | 真实列表、空/错/归档状态及当前权限 |
| `squad-editor.tsx`、`squad-members.tsx` | 资料表单、职责与维护权、历史关系及显式 leader 清除 |
| `squad-add.tsx`、`squad-invites.tsx` | 已有人员/Agent选择、管理员定向邀请与目录入口 |
| `translations.ts`、`*.test.tsx` | 中英文词汇及网络边界、创建、归档和权限回归 |

依赖 spaces 的当前租户、members 的准入表单、auth 的部署模式及 UI 原语，由路由组合根消费。后端拥有授权与事务；本模块只提交显式意图，冲突后重读。成员/插件 SSE、重连与聚焦刷新真实状态。公开邀请仅管理员挂载，归档页面保留历史且不提供写入口。MSW 测试验证真实接口请求，不用演示数据代替失败。
