package antsx

// maxSelectNum 是静态 select 优化支持的最大 channel 数量。
// 当合并的流数量 ≤ maxSelectNum 时，使用编译时展开的 select 分支，
// 避免 reflect.Select 的反射开销。超过此阈值时降级为 reflect.Select。
const maxSelectNum = 5

// receiveN 从多个 stream 的 items channel 中选择第一个就绪的进行接收。
// 通过编译时展开的函数表实现静态 select，避免 reflect.Select 的运行时开销。
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
	return []func([]int, []*stream[T]) (int, *streamItem[T], bool){
		nil,
		func(cl []int, ss []*stream[T]) (int, *streamItem[T], bool) {
			item, ok := <-ss[cl[0]].items
			return cl[0], &item, ok
		},
		func(cl []int, ss []*stream[T]) (int, *streamItem[T], bool) {
			select {
			case item, ok := <-ss[cl[0]].items:
				return cl[0], &item, ok
			case item, ok := <-ss[cl[1]].items:
				return cl[1], &item, ok
			}
		},
		func(cl []int, ss []*stream[T]) (int, *streamItem[T], bool) {
			select {
			case item, ok := <-ss[cl[0]].items:
				return cl[0], &item, ok
			case item, ok := <-ss[cl[1]].items:
				return cl[1], &item, ok
			case item, ok := <-ss[cl[2]].items:
				return cl[2], &item, ok
			}
		},
		func(cl []int, ss []*stream[T]) (int, *streamItem[T], bool) {
			select {
			case item, ok := <-ss[cl[0]].items:
				return cl[0], &item, ok
			case item, ok := <-ss[cl[1]].items:
				return cl[1], &item, ok
			case item, ok := <-ss[cl[2]].items:
				return cl[2], &item, ok
			case item, ok := <-ss[cl[3]].items:
				return cl[3], &item, ok
			}
		},
		func(cl []int, ss []*stream[T]) (int, *streamItem[T], bool) {
			select {
			case item, ok := <-ss[cl[0]].items:
				return cl[0], &item, ok
			case item, ok := <-ss[cl[1]].items:
				return cl[1], &item, ok
			case item, ok := <-ss[cl[2]].items:
				return cl[2], &item, ok
			case item, ok := <-ss[cl[3]].items:
				return cl[3], &item, ok
			case item, ok := <-ss[cl[4]].items:
				return cl[4], &item, ok
			}
		},
	}[len(chosenList)](chosenList, ss)
}
