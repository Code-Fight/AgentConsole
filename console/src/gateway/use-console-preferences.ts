import { useCallback, useEffect, useMemo, useSyncExternalStore } from "react";
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
  hasAttempted: boolean;
  hasLoadedSuccessfully: boolean;
  isLoading: boolean;
  isSaving: boolean;
  loadError: string | null;
  saveError: string | null;
}

const initialState: ConsolePreferencesState = {
  preferences: null,
  hasAttempted: false,
  hasLoadedSuccessfully: false,
  isLoading: false,
  isSaving: false,
  loadError: null,
  saveError: null,
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

function getSnapshot() {
  return store.state;
}

export function resetConsolePreferencesStoreForTests() {
  store.state = { ...initialState };
  loadPromise = null;
  emitChange();
}

interface UseConsolePreferencesOptions {
  enabled?: boolean;
}

const asyncNull = async () => null;

async function requestConsolePreferences(
  init?: RequestInit,
): Promise<ConsolePreferencesResponse> {
  const response = await fetch("/settings/console", {
    ...init,
    headers: {
      Accept: "application/json",
      ...(init?.headers ?? {}),
    },
  });

  if (!response.ok) {
    throw new Error(`Request failed with status ${response.status}`);
  }

  return (await response.json()) as ConsolePreferencesResponse;
}

export function useConsolePreferences(options?: UseConsolePreferencesOptions) {
  const enabled = options?.enabled ?? true;
  const snapshot = useSyncExternalStore(subscribe, getSnapshot, getSnapshot);

  const loadPreferences = useCallback(async () => {
    if (loadPromise) {
      return loadPromise;
    }
    setStoreState({ isLoading: true, loadError: null });
    loadPromise = (async () => {
      try {
        const response = await requestConsolePreferences();
        setStoreState({
          preferences: response.preferences,
          loadError: null,
          saveError: null,
          hasAttempted: true,
          hasLoadedSuccessfully: true,
        });
      } catch (loadError) {
        setStoreState({
          loadError:
            loadError instanceof Error
              ? loadError.message
              : "Unable to load console preferences.",
          hasAttempted: true,
          hasLoadedSuccessfully: false,
        });
      } finally {
        loadPromise = null;
        setStoreState({ isLoading: false });
      }
    })();
    return loadPromise;
  }, []);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    if (!store.state.hasAttempted && !store.state.isLoading) {
      void loadPreferences();
    }
  }, [enabled, loadPreferences]);

  const savePreferences = useCallback(
    async (next: ConsolePreferences) => {
      setStoreState({ isSaving: true, saveError: null });
      try {
        const payload: ConsolePreferencesRequest = { preferences: next };
        const response = await requestConsolePreferences({
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(payload),
        });
        setStoreState({
          preferences: response.preferences,
          saveError: null,
        });
        return response.preferences;
      } catch (saveError) {
        setStoreState({
          saveError:
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
    if (snapshot.loadError) {
      return null;
    }
    return snapshot.hasAttempted ? emptyPreferences : null;
  }, [snapshot.preferences, snapshot.loadError, snapshot.hasAttempted]);

  if (!enabled) {
    return {
      preferences: null,
      isLoading: false,
      isSaving: false,
      error: null,
      loadError: null,
      saveError: null,
      hasAttempted: false,
      hasLoadedSuccessfully: false,
      reload: asyncNull,
      savePreferences: asyncNull,
      updatePreferences: asyncNull,
    };
  }

  return {
    preferences: normalized,
    isLoading: snapshot.isLoading,
    isSaving: snapshot.isSaving,
    error: snapshot.loadError,
    loadError: snapshot.loadError,
    saveError: snapshot.saveError,
    hasAttempted: snapshot.hasAttempted,
    hasLoadedSuccessfully: snapshot.hasLoadedSuccessfully,
    reload: loadPreferences,
    savePreferences,
    updatePreferences,
  };
}
