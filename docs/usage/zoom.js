// Opens a figure's image at full size; Braille dots only read correctly at
// their own resolution, so screenshots are worth seeing unscaled.
document.addEventListener('click', (e) => {
  const img = e.target.closest('figure img')
  const open = document.querySelector('.zoom')
  if (open) {
    open.remove()
    return
  }
  if (!img) return
  const box = document.createElement('div')
  box.className = 'zoom'
  const big = document.createElement('img')
  big.src = img.currentSrc || img.src
  big.alt = img.alt
  box.append(big)
  document.body.append(box)
})
document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') document.querySelector('.zoom')?.remove()
})
