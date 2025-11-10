# 图执行器说明

本文记录 `graph.Graph` 执行流程，便于理解 `Execute` 方法与 `handleChildSignal` 的调度逻辑。

## 核心数据结构

- **`expectedParents`**：构图阶段统计出来的“某个节点拥有多少唯一父节点”。条件分支若指向同一节点只记一次。
- **`completedParents`**：运行时记录“已经向子节点报告完成”的父节点数量，哪怕该父节点本轮并未触发子节点。
- **`parentHits`**：记录真正向子节点发送执行结果的父节点数量，仅在参与执行时增加。
- **`awaiting`**：标记节点是否已排入队列，避免重复调度。
- **`queue`**：广度优先队列，驱动整个执行过程。

## `Execute` 算法

1. **预处理阶段**：调用 `buildParentCounts` 统计 `expectedParents`，为后续 fork-join 判定做准备。
2. **调度初始化**：把起始节点放入 `queue`，并初始化上述计数器。
3. **循环执行**：
   - 出队当前节点，检测是否超出 `maxVisits`。
   - 若为 `End` 节点，直接执行并返回最终状态。
   - 否则执行节点的 `Execute` 函数，更新状态。
   - 通过 `resolveNextNodes` 解析真正被激活的子节点集合 `nextNodes`，并计算该节点所有潜在子节点 `allChildren`。
   - 对 `nextNodes` 中的每个子节点调用 `handleChildSignal(..., participated=true, ...)`，表示本轮确实向其传递了 token。
   - 对 `allChildren` 中未被激活的节点调用 `handleChildSignal(..., participated=false, ...)`，告知“该父节点已完成但未触发你”，保证 `WaitAllParents` 场景不会卡住。
   - 清零当前节点在 `parentHits`、`completedParents` 中的计数，准备下一轮。
4. **终止条件**：队列耗尽即代表所有可达节点执行完毕，返回最新的 `state`。

## `handleChildSignal` 细节

- 当 `WaitAllParents=false` 时，只有 `participated=true` 才会将子节点入队；未触发的父节点对该子节点无影响。
- 当 `WaitAllParents=true` 时：
  1. 无论 `participated` 与否，`completedParents` 都会自增，确保“父节点已结束”的事实被记录。
  2. 只有 `parentHits > 0`（至少一个父节点真正触发了它）且 `completedParents == expectedParents`（全部父节点都完成）时，子节点才会入队，从而实现精准的 fork-join。

## 场景覆盖

- **条件分支汇聚**：如果条件节点的两个分支最终指向同一节点，仅命中的分支会累计 `parentHits`，未命中的分支只贡献 `completedParents`，因此下游节点不会被重复执行。
- **多父节点 + 必选父节点**：通过 `builder.RequireAllParents("D")` 可声明“D 必须等待所有父节点完成”，无论某些父亲是可选还是必选，调度器都会在满足 `completedParents == expectedParents` 后再运行 D。
- **混合结构**：若某节点既有 fork-join 父节点，也有普通父节点，调度逻辑会自动根据 `WaitAllParents` 决定是否立即运行或等待。

通过上述机制，`Graph` 可以同时支持线性流程、条件分支、fan-out/fan-in 以及多父节点的 barrier，同步逻辑全部内置在调度器中，业务方只需配置节点与边即可。
