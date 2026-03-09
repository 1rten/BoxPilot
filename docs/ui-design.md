# BoxPilot UI Notes

[中文](./zh-CN/ui-design.md)

This document records the current UI direction instead of keeping an old AI prompt.

## Current Direction

BoxPilot is closer to a lightweight operations console than a generic admin template:

- denser than a marketing site
- lighter than a traditional monitoring dashboard
- focused on runtime state, diagnostics, and quick actions

## Visual Traits

- top navigation, not a sidebar-first shell
- light theme
- card-heavy layout
- status shown through badges, tags, and dots
- data tables for subscriptions and nodes
- Dashboard acting as the operational overview

## Interaction Priorities

### Global Proxy Control

The top-bar Proxy control is a high-frequency entry point:

- start / stop forwarding
- current status
- selected node count
- business group summary
- proxy chain check

### Dashboard

Primary goal: answer whether the runtime is healthy now.

### Settings

Current design keeps save and runtime apply separate. Config changes can remain pending until the user applies and restarts.

## Language

The UI is bilingual through `tr(key, fallback)`. New text should update both locale files.
