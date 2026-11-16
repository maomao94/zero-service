package antsx

// maxSelectNum 是静态 select 优化支持的最大 channel 数量。
// 当合并的流数量 ≤ maxSelectNum 时，使用编译时展开的 select 分支，
// 避免 reflect.Select 的反射开销。超过此阈值时降级为 reflect.Select。
const maxSelectNum = 5

// receiveN 从多个 stream 的 items channel 中选择第一个就绪的进行接收。
// 通过编译时展开的 switch + select 实现静态 select，避免 reflect.Select 的运行时开销，
// 也不再像函数表方案那样每次调用都构造切片。
//
// 参数：
//   - chosenList: 活跃（未关闭）stream 在 ss 中的索引列表，len 必须在 [1, maxSelectNum] 范围内
//   - ss: 全部 stream 数组
//
// 返回：
//   - int: 被选中的 stream 在 ss 中的真实索引
//   - *streamItem: 接收到的数据项
//   - bool: true 表示成功接收；false 表示该 channel 已关闭
func receiveN[T any](chosenList []int, ss []*stream[T]) (int, *streamItem[T], bool) {
	switch len(chosenList) {
	case 1:
		idx := chosenList[0]
		item, ok := <-ss[idx].items
		return idx, &item, ok
	case 2:
		select {
		case item, ok := <-ss[chosenList[0]].items:
			return chosenList[0], &item, ok
		case item, ok := <-ss[chosenList[1]].items:
			return chosenList[1], &item, ok
		}
	case 3:
		select {
		case item, ok := <-ss[chosenList[0]].items:
			return chosenList[0], &item, ok
		case item, ok := <-ss[chosenList[1]].items:
			return chosenList[1], &item, ok
		case item, ok := <-ss[chosenList[2]].items:
			return chosenList[2], &item, ok
		}
	case 4:
		select {
		case item, ok := <-ss[chosenList[0]].items:
			return chosenList[0], &item, ok
		case item, ok := <-ss[chosenList[1]].items:
			return chosenList[1], &item, ok
		case item, ok := <-ss[chosenList[2]].items:
			return chosenList[2], &item, ok
		case item, ok := <-ss[chosenList[3]].items:
			return chosenList[3], &item, ok
		}
	case 5:
		select {
		case item, ok := <-ss[chosenList[0]].items:
			return chosenList[0], &item, ok
		case item, ok := <-ss[chosenList[1]].items:
			return chosenList[1], &item, ok
		case item, ok := <-ss[chosenList[2]].items:
			return chosenList[2], &item, ok
		case item, ok := <-ss[chosenList[3]].items:
			return chosenList[3], &item, ok
		case item, ok := <-ss[chosenList[4]].items:
			return chosenList[4], &item, ok
		}
	default:
		panic("antsx: receiveN requires 1..maxSelectNum active streams")
	}
}
