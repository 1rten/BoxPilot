# BoxPilot UI 说明

[English](../ui-design.md)

本文档记录当前 UI 的实际方向。

## 当前方向

BoxPilot 更接近轻量运维控制台，而不是通用后台模板：

- 信息密度高于营销站
- 低于传统监控面板
- 强调运行时状态、诊断和快捷操作

## 核心交互

- 顶栏 Proxy 控件负责 start / stop forwarding 和摘要展示
- Dashboard 负责回答“当前是否正常工作”
- Settings 保持“保存配置”和“应用运行时”分离
