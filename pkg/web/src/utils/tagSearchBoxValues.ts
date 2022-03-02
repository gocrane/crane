import { TagValue, AttributeValue } from 'tea-component';

export const fromTagValues = <ObjectType = object>(
  tagsValues: TagValue[],
  attributes: AttributeValue[]
): ObjectType => {
  const obj = {};

  (tagsValues ?? []).forEach(boxValues => {
    const nextValue = (boxValues.values ?? []).map(value => value.name);

    if (boxValues?.attr?.key) {
      obj[boxValues.attr.key] = nextValue;
    } else if (attributes[0]?.key) {
      obj[attributes[0]?.key] = nextValue; // 沒有選擇Key，直接複制貼上時，boxValues.attr為null，默認使用第一個AttribuesKey
    }
  });

  return obj as ObjectType;
};
export const toTagValues = <ObjectType = object>(object: ObjectType, attributes: AttributeValue[]): TagValue[] => {
  const values: TagValue[] = [];

  Object.keys(object ?? []).forEach(key => {
    for (let i = 0; i < attributes.length; i++) {
      if (attributes[i].key === key) {
        if (Array.isArray(object[key])) {
          values.push({
            attr: attributes[i],
            values: (object[key] as string[]).map(a => {
              return { key, name: a };
            })
          });
        } else if (typeof object[key] === 'string') {
          values.push({
            attr: attributes[i],
            values: [{ key, name: object[key] as string }]
          });
        }
      }
    }
  });

  return values;
};
