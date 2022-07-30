import { Color } from 'tvision-color';
import { defaultColor, darkColor, CHART_COLORS } from 'configs/color';
import { ETheme } from 'types/index.d';

/**
 * 依据主题颜色获取 ColorList
 * @param theme
 * @param themeColor
 */
function getColorFromThemeColor(theme: string, themeColor: string): Array<string> {
  let themeColorList = [];
  const isDarkMode = theme === ETheme.dark;
  const colorLowerCase = themeColor.toLocaleLowerCase();

  if (defaultColor.includes(colorLowerCase)) {
    const colorIdx = defaultColor.indexOf(colorLowerCase);
    const defaultGradients = !isDarkMode ? defaultColor : darkColor;
    const spliceThemeList = defaultGradients.slice(0, colorIdx);
    themeColorList = defaultGradients.slice(colorIdx, defaultGradients.length).concat(spliceThemeList);
  } else {
    themeColorList = Color.getRandomPalette({
      color: themeColor,
      colorGamut: 'bright',
      number: 8,
    });
  }

  return themeColorList;
}

/**
 *
 * @param theme 当前主题
 * @param themeColor 当前主题色
 */
export function getChartColor(theme: ETheme, themeColor: string) {
  const colorList = getColorFromThemeColor(theme, themeColor);
  // 图表颜色
  const chartColors = CHART_COLORS[theme];
  return { ...chartColors, colorList };
}
