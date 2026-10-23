-- A hint while the dragon sleeps and the knight is close.
fields = {
  reach = range(14, 2, 40),
  hint = "press E",
}

function update(self, dt)
  local hero = find("hero")
  local dragon = self:component("df.dragon")
  if hero and not dragon.awake and self:distance(hero) < self.reach then
    say(self.hint, 0.4)
  end
end
