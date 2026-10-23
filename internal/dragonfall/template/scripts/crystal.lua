-- The moon crystal greets the knight and glows brighter up close.
function update(self, dt)
  local hero = find("hero")
  if not hero then return end
  if dist(self, hero) < 6 then
    say("the moon remembers", 0.5)
    self:set("glow", 1)
  else
    self:set("glow", 0.6)
  end
end
