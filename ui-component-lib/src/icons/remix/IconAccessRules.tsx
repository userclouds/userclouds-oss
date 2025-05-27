import { IconTypes, getSize } from '../iconHelpers';

function IconAccessRules({
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
      <title>Access Rules</title>
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M5 0L4 1V3H12V1L11 0H5ZM3 2H1C0 2 0 3 0 3V18C0 18 0 19 1 19H15C16 19 16 18 16 18V3C16 3 16 2 15 2H13V4H3V2ZM7 15H14V16H7V15ZM5 14H2V17H5V14ZM7 11H14V12H7V11ZM5 10H2V13H5V10ZM7 7H14V8H7V7ZM5 6H2V9H5V6ZM4 11H3V12H4V11ZM4 15H3V16H4V15ZM4 7H3V8H4V7Z"
        fill="#283354"
      />
    </svg>
  );
}

export default IconAccessRules;
