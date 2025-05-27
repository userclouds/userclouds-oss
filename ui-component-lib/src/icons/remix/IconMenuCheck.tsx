import { IconTypes, getSize } from '../iconHelpers';

function IconMenuCheck({ className, size = 'medium' }: IconTypes): JSX.Element {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={getSize(size)}
      height={getSize(size)}
      className={className}
      fill="currentColor"
    >
      <title>Menu Check</title>
      <path
        d="M8.33336 12.6434L15.9934 4.98254L17.1725 6.16088L8.33336 15L3.03003 9.69671L4.20836 8.51838L8.33336 12.6434Z"
        fill="#283354"
      />
    </svg>
  );
}

export default IconMenuCheck;
