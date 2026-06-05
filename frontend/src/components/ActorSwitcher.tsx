import { useActor } from "../state/actor";
import { useActors } from "../hooks/queries";

// Lets you "become" any seeded actor (tanaka / acc-bot / accountant) to drive
// the multi-actor flow without real auth: e.g. switch to the accountant to
// clear an approval gate.
export function ActorSwitcher() {
  const { actorId, setActorId } = useActor();
  const actors = useActors();
  return (
    <div className="actor-switcher" style={{ display: "flex", alignItems: "center", gap: 8 }}>
      <span className="muted" style={{ fontSize: 12 }}>担当者:</span>
      <select value={actorId} onChange={(e) => setActorId(e.target.value)} style={{ width: "auto" }}>
        {(actors.data ?? [{ id: actorId, display_name: actorId, kind: "human", roles: [], capabilities: [] }]).map(
          (a) => (
            <option key={a.id} value={a.id}>
              {a.display_name}（{a.kind === "agent" ? "エージェント" : a.roles.join(",") || "人"}）
            </option>
          ),
        )}
      </select>
    </div>
  );
}
