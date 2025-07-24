export enum K8SUNIT {
  m = 'm',
  /** 核 */
  unit = 'unit',
  K = 'k',
  M = 'M',
  G = 'G',
  T = 'T',
  P = 'P',
  E = 'E',
  Ki = 'Ki',
  Mi = 'Mi',
  Gi = 'Gi',
  Ti = 'Ti',
  Pi = 'Pi',
  Ei = 'Ei',
}

const UNITS1024 = [K8SUNIT.m, K8SUNIT.unit, K8SUNIT.Ki, K8SUNIT.Mi, K8SUNIT.Gi, K8SUNIT.Ti, K8SUNIT.Pi, K8SUNIT.Ei];
const UNITS1000 = [K8SUNIT.m, K8SUNIT.unit, K8SUNIT.K, K8SUNIT.M, K8SUNIT.G, K8SUNIT.T, K8SUNIT.P, K8SUNIT.E];

const UNITS1024_MAP_TO_UNITS1000: Partial<Record<K8SUNIT, K8SUNIT>> = {
  [K8SUNIT.Ki]: K8SUNIT.K,
  [K8SUNIT.Mi]: K8SUNIT.M,
  [K8SUNIT.Gi]: K8SUNIT.G,
  [K8SUNIT.Ti]: K8SUNIT.T,
  [K8SUNIT.Pi]: K8SUNIT.P,
  [K8SUNIT.Ei]: K8SUNIT.E,
};

const UNITS1000_MAP_TO_UNITS1024: Partial<Record<K8SUNIT, K8SUNIT>> = {
  [K8SUNIT.K]: K8SUNIT.Ki,
  [K8SUNIT.M]: K8SUNIT.Mi,
  [K8SUNIT.G]: K8SUNIT.Gi,
  [K8SUNIT.T]: K8SUNIT.Ti,
  [K8SUNIT.P]: K8SUNIT.Pi,
  [K8SUNIT.E]: K8SUNIT.Ei,
};

/**
 * 进行单位换算
 * 实现k8s数值各单位之间的相互转换
 * @param {string} value
 * @param {number} step
 * @param {number} toFixed
 */
function transformField(_value: string, step: number, toFixed: number, units: K8SUNIT[], targetUnit: K8SUNIT) {
  const reg = /^(\d+(\.\d{1,5})?)([A-Za-z]+)?$/;
  let value;
  let unitBase;
  if (reg.test(_value)) {
    [value, unitBase] = [+RegExp.$1, RegExp.$3];
    if (unitBase === '') {
      unitBase = K8SUNIT.unit;
    }
  } else {
    return '0';
  }

  // 由于m到单位1是1000进制
  const mIndex = units.indexOf(K8SUNIT.m);
  let i = units.indexOf(unitBase as K8SUNIT);
  let targetI = units.indexOf(targetUnit);
  if (step) {
    if (targetI >= i) {
      while (i < targetI) {
        value /= i + 1 <= mIndex ? 1000 : step;
        i += 1;
      }
    } else {
      while (targetI < i) {
        value *= i - 1 <= mIndex ? 1000 : step;
        targetI += 1;
      }
    }
  }

  let svalue;
  if (value > 1) {
    svalue = value.toFixed(toFixed);
    svalue = svalue.replace(/0+$/, '');
    svalue = svalue.replace(/\.$/, '');
  } else if (value) {
    // 如果数值很小，保留toFixed位有效数字
    let tens = 0;
    let v = Math.abs(value);
    while (v < 1) {
      v *= 10;
      tens += 1;
    }
    svalue = value.toFixed(tens + toFixed - 1);
    svalue = svalue.replace(/0+$/, '');
    svalue = svalue.replace(/\.$/, '');
  } else {
    svalue = value;
  }
  return String(svalue);
}

function valueTrans1000(value: string, targetUnit: K8SUNIT) {
  return transformField(value, 1000, 3, UNITS1000, targetUnit);
}

function valueTrans1024(value: string, targetUnit: K8SUNIT) {
  return transformField(value, 1024, 3, UNITS1024, targetUnit);
}

export function transformK8sUnit(data: string, targetUnit: K8SUNIT) {
  let finalData = 0;
  const requestValue: string = data ?? '';
  const reg = /^(\d+(\.\d{1,2})?)([A-Za-z]+)?$/;
  let unitBase;

  if (reg.test(requestValue)) {
    unitBase = RegExp.$3;
  }

  const isBase1000 = unitBase ? UNITS1000.includes(unitBase as K8SUNIT) : false;
  const isTarget1000 = UNITS1000.includes(targetUnit);

  if (requestValue) {
    if (isBase1000 && isTarget1000) {
      // 1G 轉 M
      finalData = +valueTrans1000(requestValue, targetUnit);
    } else if (!isBase1000 && !isTarget1000) {
      // 1Gi 轉 Mi
      finalData = +valueTrans1024(requestValue, targetUnit);
    } else if (isBase1000 && !isTarget1000) {
      // 1G 轉 Mi
      const newTargetUnit = UNITS1024_MAP_TO_UNITS1000[targetUnit];
      if (newTargetUnit) {
        finalData = +(+valueTrans1000(requestValue, newTargetUnit) / 1.024).toFixed(3);
      } else {
        // targetUnit為m 或 unit
        finalData = +valueTrans1000(requestValue, targetUnit);
      }
    } else if (!isBase1000 && isTarget1000) {
      // 1Gi 轉 M
      const newTargetUnit = UNITS1000_MAP_TO_UNITS1024[targetUnit]; // 先將Gi轉成Mi
      if (newTargetUnit) {
        finalData = +(+valueTrans1024(requestValue, newTargetUnit) * 1.024).toFixed(3);
      } else {
        finalData = +valueTrans1024(requestValue, targetUnit);
      }
    }
  } else {
    finalData = 0;
  }

  return finalData;
}
