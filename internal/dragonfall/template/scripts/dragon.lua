-- A hint while the dragon sleeps and the knight is close.
function update(self, dt)
  local hero = find("hero")
  if hero and not self:get("awake") and dist(self, hero) < 14 then
    say("press E", 0.4)
  end
end
