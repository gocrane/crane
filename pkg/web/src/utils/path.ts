/**
 * 解析当前菜单对应的路由
 * @param path1
 * @param path2
 */
export const resolve = (path1 = '', path2 = '') => {
  let separator = '/';
  if (path1.endsWith('/') || path2.startsWith('/')) {
    separator = '';
  }
  return `${path1}${separator}${path2}`;
};
