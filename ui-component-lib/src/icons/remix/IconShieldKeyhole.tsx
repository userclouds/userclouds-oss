import { IconTypes, getSize } from '../iconHelpers';

function IconShieldKeyhole({
  className,
  size = 'medium',
}: IconTypes): JSX.Element {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={getSize(size)}
      height={getSize(size)}
      className={className}
      fill="currentColor"
    >
      <title>Shield Keyhole</title>
      <path d="M12 1l8.217 1.826a1 1 0 0 1 .783.976v9.987a6 6 0 0 1-2.672 4.992L12 23l-6.328-4.219A6 6 0 0 1 3 13.79V3.802a1 1 0 0 1 .783-.976L12 1zm0 6a2 2 0 0 0-1 3.732V15h2l.001-4.268A2 2 0 0 0 12 7z" />
    </svg>
  );
}

export default IconShieldKeyhole;
