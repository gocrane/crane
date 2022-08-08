interface IOption {
  value: number | string;
  label: string;
}

// 推荐类型

export const RECOMMENDATION_RULE_TYPE_OPTIONS: Array<IOption> = [
  { value: 'Resource', label: 'Resource' },
  { value: 'Replicas', label: 'Replicas' },
];
