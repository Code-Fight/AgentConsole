import { useCallback, useEffect, useMemo, useState } from "react";
import { http } from "../common/api/http";
import type {
  ConsolePreferences,
  ConsolePreferencesRequest,
  ConsolePreferencesResponse,
} from "../common/api/types";

const emptyPreferences: ConsolePreferences = {
  consoleUrl: "",
  apiKey: "",
  profile: "",
  safetyPolicy: "",
  lastThreadId: "",
};

interface ConsolePreferencesState {
  preferences: ConsolePreferences | null;
  hasLoaded: boolean;
  isLoading: boolean;
  isSaving: boolean;
  error: string | null;
}

const initialState: ConsolePreferencesState = {
  preferences: null,
  hasLoaded: false,
  isLoading: true,
  isSaving: false,
  error: null,
};

const store = {
  state: initialState,
  listeners: new Set<() => void>(),
};

let loadPromise: Promise<void> | null = null;

function emitChange() {
  store.listeners.forEach((listener) => listener());
}

function setStoreState(patch: Partial<ConsolePreferencesState>) {
  store.state = { ...store.state, ...patch };
  emitChange();
}

function subscribe(listener: () => void) {
  store.listeners.add(listener);
  return () => {
    store.listeners.delete(listener);
  };
}

export function resetConsolePreferencesStoreForTests() {
  store.state = { ...initialState };
  loadPromise = null;
  emitChange();
}

export function useConsolePreferences() {
  const [snapshot, setSnapshot] = useState(store.state);

  useEffect(() => subscribe(() => setSnapshot(store.state)), []);

  const loadPreferences = useCallback(async () => {
    if (loadPromise) {
      return loadPromise;
    }
    setStoreState({ isLoading: true });
    loadPromise = (async () => {
      try {
        const response = await http<ConsolePreferencesResponse>("/settings/console");
        setStoreState({
          preferences: response.preferences,
          error: null,
          hasLoaded: true,
        });
      } catch (loadError) {
        setStoreState({
          error:
            loadError instanceof Error
              ? loadError.message
              : "Unable to load console preferences.",
        });
      } finally {
        loadPromise = null;
        setStoreState({ isLoading: false });
      }
    })();
    return loadPromise;
  }, []);

  useEffect(() => {
    if (!store.state.hasLoaded) {
      void loadPreferences();
    }
  }, [loadPreferences]);

  const savePreferences = useCallback(
    async (next: ConsolePreferences) => {
      setStoreState({ isSaving: true });
      try {
        const payload: ConsolePreferencesRequest = { preferences: next };
        const response = await http<ConsolePreferencesResponse>("/settings/console", {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(payload),
        });
        setStoreState({
          preferences: response.preferences,
          error: null,
          hasLoaded: true,
        });
        return response.preferences;
      } catch (saveError) {
        setStoreState({
          error:
            saveError instanceof Error
              ? saveError.message
              : "Unable to save console preferences.",
        });
        return null;
      } finally {
        setStoreState({ isSaving: false });
      }
    },
    [],
  );

  const updatePreferences = useCallback(
    async (patch: Partial<ConsolePreferences>) => {
      const base = store.state.preferences ?? emptyPreferences;
      return savePreferences({ ...base, ...patch });
    },
    [savePreferences],
  );

  const normalized = useMemo(() => {
    if (snapshot.preferences) {
      return snapshot.preferences;
    }
    if (snapshot.error) {
      return null;
    }
    return snapshot.hasLoaded ? emptyPreferences : null;
  }, [snapshot.preferences, snapshot.error, snapshot.hasLoaded]);

  return {
    preferences: normalized,
    isLoading: snapshot.isLoading,
    isSaving: snapshot.isSaving,
    error: snapshot.error,
    hasLoaded: snapshot.hasLoaded,
    reload: loadPreferences,
    savePreferences,
    updatePreferences,
  };
}
