-- The moon crystal greets the knight and glows brighter up close.
-- Fields show up in the Inspector; drop the hero there or leave it empty.
fields = {
  hero = object(),
  reach = range(6, 1, 20),
  message = "the moon remembers",
}

function update(self, dt)
  local hero = self.hero or find("hero")
  if not hero then return end
  local glow = self:component("blob")
  if self:distance(hero) < self.reach then
    say(self.message, 0.5)
    glow.glow = 1
  else
    glow.glow = 0.6
  end
end
