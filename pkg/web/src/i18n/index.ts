import i18n, { t } from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import ICU from 'i18next-icu';

import { initReactI18next, Trans } from 'react-i18next';

import zh from './resources/zh/translation.json';

const resources = {
  zh: {
    translation: { ...zh }
  }
};

export enum SupportLanguages {
  zh = 'zh',
  en = 'en'
}

i18n
  .use(LanguageDetector)
  .use(new ICU())
  .use(initReactI18next) // passes i18n down to react-i18next
  .init({
    fallbackLng: 'zh',
    resources,
    debug: process.env.NODE_ENV === 'development',
    saveMissing: true,

    interpolation: {
      escapeValue: false // react already safes from xss
    }
  });

const changeLanguage = (language: SupportLanguages) => {
  i18n.changeLanguage(language);
};

export { t, Trans, changeLanguage };

export default i18n;
