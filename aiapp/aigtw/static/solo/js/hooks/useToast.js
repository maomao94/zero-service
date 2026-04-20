import { useCallback, useMemo, useState } from "../lib/deps.js";

let nextID = 1;

export function useToasts() {
  const [items, setItems] = useState([]);
  const push = useCallback((msg, kind = "info", ttl = 3000) => {
    const id = nextID++;
    setItems((s) => [...s, { id, msg, kind }]);
    setTimeout(() => setItems((s) => s.filter((x) => x.id !== id)), ttl);
  }, []);
  const remove = useCallback((id) => {
    setItems((s) => s.filter((x) => x.id !== id));
  }, []);
  // 关键: 返回对象的引用与 items 绑定, 没 toast 时稳定, 避免上游依赖它的
  // useCallback / useEffect 每次 render 都重建 (会导致 /modes /sessions 被反复拉).
  return useMemo(() => ({ items, push, remove }), [items, push, remove]);
}
