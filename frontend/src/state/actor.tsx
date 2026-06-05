import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { useQueryClient } from "@tanstack/react-query";

type ActorCtx = {
  actorId: string;
  setActorId: (id: string) => void;
};

const Ctx = createContext<ActorCtx>({ actorId: "tanaka", setActorId: () => {} });

export function ActorProvider({ children }: { children: ReactNode }) {
  const qc = useQueryClient();
  const [actorId, setActorIdState] = useState<string>(
    () => localStorage.getItem("actorId") || "tanaka",
  );
  const setActorId = useCallback(
    (id: string) => {
      localStorage.setItem("actorId", id);
      setActorIdState(id);
      // Re-fetch everything as the new actor (different inbox, permissions).
      qc.invalidateQueries();
    },
    [qc],
  );
  const value = useMemo(() => ({ actorId, setActorId }), [actorId, setActorId]);
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export const useActor = () => useContext(Ctx);
