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

export function useConsolePreferences() {
  const [preferences, setPreferences] = useState<ConsolePreferences | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadPreferences = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await http<ConsolePreferencesResponse>("/settings/console");
      setPreferences(response.preferences);
      setError(null);
    } catch (loadError) {
      setError(
        loadError instanceof Error ? loadError.message : "Unable to load console preferences.",
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadPreferences();
  }, [loadPreferences]);

  const savePreferences = useCallback(
    async (next: ConsolePreferences) => {
      setIsSaving(true);
      try {
        const payload: ConsolePreferencesRequest = { preferences: next };
        const response = await http<ConsolePreferencesResponse>("/settings/console", {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(payload),
        });
        setPreferences(response.preferences);
        setError(null);
        return response.preferences;
      } catch (saveError) {
        setError(
          saveError instanceof Error ? saveError.message : "Unable to save console preferences.",
        );
        return null;
      } finally {
        setIsSaving(false);
      }
    },
    [],
  );

  const updatePreferences = useCallback(
    async (patch: Partial<ConsolePreferences>) => {
      const base = preferences ?? emptyPreferences;
      return savePreferences({ ...base, ...patch });
    },
    [preferences, savePreferences],
  );

  const normalized = useMemo(
    () => preferences ?? (isLoading ? null : emptyPreferences),
    [preferences, isLoading],
  );

  return {
    preferences: normalized,
    isLoading,
    isSaving,
    error,
    reload: loadPreferences,
    savePreferences,
    updatePreferences,
  };
}
