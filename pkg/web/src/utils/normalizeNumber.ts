export const normalizeNumber = (num: number) => {
  if (num < 0.01) {
    return num.toPrecision(1);
  } else {
    return num.toFixed(2);
  }
};
