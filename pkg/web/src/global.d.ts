declare module '*.avif' {
  export default src as string;
}

declare module '*.bmp' {
  export default src as string;
}

declare module '*.gif' {
  export default src as string;
}

declare module '*.jpg' {
  export default src as string;
}

declare module '*.jpeg' {
  export default src as string;
}

declare module '*.png' {
  export default src as string;
}

declare module '*.webp' {
  export default src as string;
}

declare module '*.svg' {
  export default src as string;
}
declare module '*.svg?component' {
  export default src as string;
}
declare module '*.module.css' {
  export default classes as { readonly [key: string]: string };
}

declare module '*.module.less' {
  export default classes as { readonly [key: string]: string };
}

declare module '*.less' {
  export default classes as { readonly [key: string]: string };
}

declare module 'hex-to-hsl';

declare interface ImportMeta {
  env: {
    MODE: 'development' | 'test' | 'production';
  };
}
