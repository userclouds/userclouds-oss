export interface IconTypes {
  size?: 'medium' | 'small' | 'large';
  className?: string;
}

export function getSize(size: string) {
  const sizes = {
    large: 32,
    medium: 20,
    small: 16,
  };

  return sizes[size];
}
