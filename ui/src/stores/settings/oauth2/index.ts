import { produce } from "immer";
import { create } from "zustand";

import { type OAuth2SettingsContent, SETTINGS_NAMES, type SettingsModel } from "@/domain/settings";
import { get as getSettings, save as saveSettings } from "@/repository/settings";

interface OAuth2SettingsState {
  settings: OAuth2SettingsContent;
  loading: boolean;
  loadedAtOnce: boolean;
}

interface OAuth2SettingsStore extends OAuth2SettingsState {
  loadSettings: (refresh?: boolean) => Promise<void>;
  saveSettings: (settings: OAuth2SettingsContent) => Promise<void>;
}

export const useOAuth2SettingsStore = create<OAuth2SettingsStore>((set, get) => {
  let fetcher: Promise<SettingsModel<OAuth2SettingsContent>> | null = null;
  let model: SettingsModel<OAuth2SettingsContent>;

  return {
    settings: { providers: [] },
    loading: false,
    loadedAtOnce: false,

    loadSettings: async (refresh = true) => {
      if (!refresh && get().loadedAtOnce) {
        return;
      }

      fetcher ??= getSettings(SETTINGS_NAMES.OAUTH2);

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
      model ??= await getSettings(SETTINGS_NAMES.OAUTH2);
      model = await saveSettings<OAuth2SettingsContent>({
        ...model,
        content: settings,
      });

      set(
        produce((state: OAuth2SettingsState) => {
          state.settings = model.content;
          state.loadedAtOnce = true;
        })
      );
    },
  };
});
