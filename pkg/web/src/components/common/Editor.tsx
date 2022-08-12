import React, { useEffect, useRef } from 'react';

import 'monaco-editor/esm/vs/basic-languages/dockerfile/dockerfile.contribution';
import 'monaco-editor/esm/vs/basic-languages/yaml/yaml.contribution';
import MonacoEditor, { MonacoEditorProps } from 'react-monaco-editor';

export interface EditorProps {
  value: string;
}

const Editor = (props: EditorProps) => {
  const { value } = props;
  const options: Partial<MonacoEditorProps['options']> = {
    readOnly: true,
    theme: 'vs-dark',
    fontSize: 14,
    formatOnType: true,
    wordWrap: 'on',
  };

  const ref = useRef(null);

  return <MonacoEditor value={value} language={'yaml'} height={700} options={options} ref={ref} />;
};

export default Editor;
