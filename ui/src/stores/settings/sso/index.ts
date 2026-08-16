import { produce } from "immer";
import { create } from "zustand";

import { SETTINGS_NAMES, type SSOSettingsContent, type SettingsModel } from "@/domain/settings";
import { get as getSettings, save as saveSettings } from "@/repository/settings";

interface SSOSettingsState {
  settings: SSOSettingsContent;
  loading: boolean;
  loadedAtOnce: boolean;
}

interface SSOSettingsStore extends SSOSettingsState {
  loadSettings: (refresh?: boolean) => Promise<void>;
  saveSettings: (settings: SSOSettingsContent) => Promise<void>;
}

export const useSSOSettingsStore = create<SSOSettingsStore>((set, get) => {
  let fetcher: Promise<SettingsModel<SSOSettingsContent>> | null = null;
  let model: SettingsModel<SSOSettingsContent>;

  return {
    settings: {},
    loading: false,
    loadedAtOnce: false,

    loadSettings: async (refresh = true) => {
      if (!refresh && get().loadedAtOnce) {
        return;
      }

      fetcher ??= getSettings(SETTINGS_NAMES.SSO);

      try {
        set({ loading: true });
        model = await fetcher;
        set({ settings: model.content, loadedAtOnce: true });
      } finally {
        fetcher = null;
        set({ loading: false });
      }
    },

    saveSettings: async (settings) => {
      model ??= await getSettings(SETTINGS_NAMES.SSO);
      model = await saveSettings<SSOSettingsContent>({
        ...model,
        content: settings,
      });

      set(
        produce((state: SSOSettingsState) => {
          state.settings = model.content;
          state.loadedAtOnce = true;
        })
      );
    },
  };
});
