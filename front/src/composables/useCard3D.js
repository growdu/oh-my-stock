// useCard3D —— 鼠标跟随 3D 倾斜 + 卡片内部 translateZ 分层
// 用法：
//   const { onTiltMove, onTiltLeave } = useCard3D({ max: 12 })
//   <div class="stock-card" @mousemove="onTiltMove" @mouseleave="onTiltLeave"> ... </div>
export function useCard3D(opts = {}) {
  const max    = opts.max    ?? 10   // 倾斜最大角度（度）
  const depth  = opts.depth  ?? 14   // 鼠标离开时 translateZ 抬起
  const ease   = opts.ease   ?? 'cubic-bezier(.2,.7,.3,1)'

  const onTiltMove = (e) => {
    const el = e.currentTarget
    if (!el) return
    const r = el.getBoundingClientRect()
    const px = (e.clientX - r.left) / r.width   // 0..1
    const py = (e.clientY - r.top)  / r.height  // 0..1
    const ry = (px - 0.5) * 2 * max
    const rx = -(py - 0.5) * 2 * max
    el.style.transform =
      `perspective(900px) rotateX(${rx.toFixed(2)}deg) rotateY(${ry.toFixed(2)}deg) translateZ(0)`
    el.style.transition = `transform .12s ${ease}`
    // 同步内层高光（可选）：根据鼠标位置给阴影加点方向感
    el.style.boxShadow =
      `${(-ry/4).toFixed(2)}px ${(rx/4).toFixed(2)}px 18px rgba(64,158,255,.22),` +
      `0 6px 18px rgba(0,0,0,.08)`
  }

  const onTiltLeave = (e) => {
    const el = e.currentTarget
    if (!el) return
    el.style.transition = `transform .45s ${ease}, box-shadow .35s ${ease}`
    el.style.transform = ''
    el.style.boxShadow = ''
  }

  return { onTiltMove, onTiltLeave }
}
