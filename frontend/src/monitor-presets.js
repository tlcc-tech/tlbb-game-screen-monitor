export const PRESET_ORDER = [
  { key: "declare_war", label: "有人宣战", category: "combat" },
  { key: "under_attack", label: "受到攻击", category: "combat" },
  { key: "death", label: "死亡", category: "status" },
  { key: "disconnect", label: "掉线", category: "network" },
];

export function groupTemplates(templates) {
  const builtins = [];
  const customs = [];
  for (const t of templates || []) {
    if (t.presetKey) {
      builtins.push(t);
    } else {
      customs.push(t);
    }
  }
  builtins.sort(
    (a, b) =>
      PRESET_ORDER.findIndex((p) => p.key === a.presetKey) -
      PRESET_ORDER.findIndex((p) => p.key === b.presetKey),
  );
  return { builtins, customs };
}

export function findBuiltin(templates, presetKey) {
  return (templates || []).find((t) => t.presetKey === presetKey);
}
