import { createContext, useContext, type ReactNode } from "react";
import type { ThreadHubViewModel } from "../hooks/use-thread-hub";

const ThreadHubContext = createContext<ThreadHubViewModel | null>(null);

export function ThreadHubProvider(props: {
  value: ThreadHubViewModel;
  children: ReactNode;
}) {
  return (
    <ThreadHubContext.Provider value={props.value}>
      {props.children}
    </ThreadHubContext.Provider>
  );
}

export function useThreadHubContext() {
  return useContext(ThreadHubContext);
}
