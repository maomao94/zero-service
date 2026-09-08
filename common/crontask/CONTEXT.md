# Scheduling Context

Scheduling 上下文描述可被周期或人工触发的任务，以及每次调度所对应的时间身份。它不定义具体业务任务的内容。

## Language

**Scheduled Time**:
当前执行最初计划发生的时间；同一次计划执行发生重试时保持不变。
_Avoid_: Next Run, Last Run

**Next Run**:
任务空闲时的下次计划时间；任务被领取后，该值暂时代表本次领取的租约截止时间。
_Avoid_: Scheduled Time

**Last Run**:
任务最近一次成功处理的实际完成时间。
_Avoid_: Last Scheduled Run

**Last Scheduled Run**:
最近一次成功周期执行的原计划时间；人工执行不会改变它。
_Avoid_: Last Run

**Task Code**:
业务方为可调度任务指定的稳定编码。
_Avoid_: Job ID, Trace ID
