---
title: 存储引擎
description: 采用 Sqlite + Pebble 的组合存储架构，提供关系型数据、高性能索引和可选缓存层。
order: 30
section: core-concepts
tags: [storage, sqlite, pebble, redis, database]
category: docs
---

# 存储

采用 Sqlite + Pebble 的组合：

- Sqlite 负责关系型数据与事务，轻量可靠。
- Pebble 负责高性能键值索引，适合热点读写与时间序列。
- 可选 Redis 作为缓存层，在热点场景下降低延迟。