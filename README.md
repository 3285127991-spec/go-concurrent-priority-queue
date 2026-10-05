# go-concurrent-priority-queue

纯 Go 实现的**并发安全有界优先队列**后端库，附带 CLI 演示与自动化测试。
仅依赖 Go 标准库，无数据库、无网络服务、无第三方依赖。

## 特性

- 创建时指定最大容量，任何时刻元素数量不超过容量
- 元素为 `{ID, Priority, Value}`：ID 为唯一非空字符串，Priority 为 int，Value 为字符串
- 出队顺序：优先级从高到低；同优先级按最初成功 Push 的先后 FIFO
- `UpdatePriority` 只改优先级，不改变同优先级间的 FIFO 顺序
- 所有操作 goroutine 安全；返回值均为独立副本，调用方修改不影响内部状态
- 基于二叉堆 + ID 索引：Push/Pop/UpdatePriority/Remove 均为 O(log n)，Get 为 O(1)

## 项目结构

| 文件 | 说明 |
| --- | --- |
| `queue.go` | 公开 API：Queue、Item、错误定义 |
| `heap.go` | 内部 `container/heap` 实现（最大堆，序号打破平局） |
| `cmd/demo/main.go` | CLI 演示程序 |
| `queue_test.go` | 功能测试 |
| `queue_concurrent_test.go` | 并发与不变量测试 |

## API

```go
q, err := priorityqueue.New(128)        // 容量必须为正，否则 ErrInvalidCapacity

err = q.Push("a", 5, "hello")           // ErrEmptyID / ErrDuplicateID / ErrQueueFull
it, ok := q.Get("a")                    // 按 ID 查询，ok=false 表示不存在
ok = q.UpdatePriority("a", 9)           // false 表示 ID 不存在
ok = q.Remove("a")                      // false 表示 ID 不存在
it, ok = q.Peek()                       // 查看最高优先级元素，不删除
it, ok = q.Pop()                        // 取出并删除最高优先级元素
items := q.Snapshot()                   // 全部元素的有序副本（出队顺序）
n := q.Len()                            // 当前元素数量
c := q.Capacity()                       // 最大容量
```

## 设计要点

- **堆 + 索引**：`entryHeap` 实现 `container/heap.Interface`，`map[string]*entry`
  提供按 ID 的 O(1) 定位，配合 `heap.Fix`/`heap.Remove` 实现高效更新与删除，
  无需对完整切片重新排序。
- **FIFO 平局**：每个元素在 Push 成功时分配单调递增序号 `seq`，比较函数为
  `(优先级降序, seq 升序)`。`UpdatePriority` 不触碰 `seq`，因此不改变同优先级顺序。
- **并发模型**：一把 `sync.Mutex` 保护全部内部状态，无锁外共享指针。
- **副本隔离**：`Item` 是纯值类型，`Get`/`Peek`/`Pop`/`Snapshot` 均返回拷贝；
  `Snapshot` 先复制再排序，不干扰堆结构。

## 运行

需要 Go 1.27+（Windows / Linux / macOS 均可）。

```powershell
# 运行全部测试
go test ./... -count=1

# 使用 race detector 验证并发安全（Windows 需安装 gcc）
go test -race ./... -count=1

# 运行 CLI 演示
go run ./cmd/demo
```

## 测试覆盖

- 基本入队/出队、空队列 Pop/Peek
- 优先级降序、同优先级 FIFO、UpdatePriority 后 FIFO 保持
- 重复 ID 拒绝、容量边界（满时仅拒绝新增，其余操作正常）
- UpdatePriority / Remove 对不存在 ID 的明确结果
- Snapshot 稳定顺序、可重复性、不影响队列
- Get/Peek/Pop/Snapshot 返回值的副本隔离
- 非法参数（非正容量、空 ID）
- 并发混合读写下的容量与唯一性不变量、并发排空不丢不重
- 随机交错 Push/Pop 与朴素参考模型对拍（5000 步）
