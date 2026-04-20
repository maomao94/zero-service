/**
 * 与 common/einox/tool/builtin/compute.go 中 evalArith 一致的纯本地求值，
 * 用于 Solo 计算器面板预览与校验（不请求服务端）。
 */

function peek(p) {
  return p.pos < p.src.length ? p.src[p.pos] : "\0";
}

function expression(p) {
  let r = term(p);
  if (r.err) return r;
  let v = r.v;
  for (;;) {
    const c = peek(p);
    if (c !== "+" && c !== "-") return { v };
    const op = c;
    p.pos += 1;
    r = term(p);
    if (r.err) return r;
    v = op === "+" ? v + r.v : v - r.v;
  }
}

function term(p) {
  let r = factor(p);
  if (r.err) return r;
  let v = r.v;
  for (;;) {
    const c = peek(p);
    if (c !== "*" && c !== "/") return { v };
    const op = c;
    p.pos += 1;
    r = factor(p);
    if (r.err) return r;
    if (op === "*") v *= r.v;
    else {
      if (r.v === 0) return { err: "calculator: div by zero" };
      v /= r.v;
    }
  }
}

function factor(p) {
  if (peek(p) === "(") {
    p.pos += 1;
    const inner = expression(p);
    if (inner.err) return inner;
    if (peek(p) !== ")") return { err: "calculator: missing ')'" };
    p.pos += 1;
    return { v: inner.v };
  }
  if (peek(p) === "-") {
    p.pos += 1;
    const inner = factor(p);
    if (inner.err) return inner;
    return { v: -inner.v };
  }

  const start = p.pos;
  while (p.pos < p.src.length) {
    const c = p.src[p.pos];
    if ((c >= "0" && c <= "9") || c === ".") p.pos += 1;
    else break;
  }
  if (start === p.pos) return { err: `calculator: expected number at ${p.pos}` };

  const slice = p.src.slice(start, p.pos);
  const n = Number(slice);
  if (!Number.isFinite(n)) return { err: "calculator: invalid number" };
  return { v: n };
}

/**
 * @param {string} raw
 * @returns {{ ok: true, value: number } | { ok: false, error: string }}
 */
export function evalArith(raw) {
  const src = String(raw).replace(/\s/g, "");
  const p = { src, pos: 0 };
  if (!src) return { ok: false, error: "calculator: expected number at 0" };

  const r = expression(p);
  if (r.err) return { ok: false, error: r.err };
  if (p.pos !== src.length) {
    const c = src[p.pos];
    return { ok: false, error: `calculator: unexpected char '${c}' at ${p.pos}` };
  }
  return { ok: true, value: r.v };
}
