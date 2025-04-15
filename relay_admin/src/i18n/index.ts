import enMessages from '../locales/en.json'
import zhMessages from '../locales/zh.json'
import { createI18n } from 'vue-i18n'
type MessageSchema = typeof enMessages;

const userLang = navigator.language
let defaultLocale = 'en';

const initalizeI18n = () => {
  if (localStorage.getItem('locale')) {
    defaultLocale = localStorage.getItem('locale')!;
  } else {
    defaultLocale = userLang.startsWith('zh') ? 'zh' : 'en'
    localStorage.setItem('locale', defaultLocale)
  }
}

console.log("defaultLocale", defaultLocale);

initalizeI18n()
const i18n = createI18n<[MessageSchema], 'en' | 'zh'>({
  legacy: false,
  locale: defaultLocale,
  fallbackLocale: 'en',
  messages: {
    zh: zhMessages,
    en: enMessages
  }
});
export default i18n;
