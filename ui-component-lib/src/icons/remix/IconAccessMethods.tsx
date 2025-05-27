import { IconTypes, getSize } from '../iconHelpers';

function IconAccessMethods({
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
      <title>Access Methods</title>
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M0 2C0 0.895386 0.895447 0 2 0H16C17.1046 0 18 0.895386 18 2V16C18 17.1046 17.1046 18 16 18H2C0.895447 18 0 17.1046 0 16V2ZM5 9.5C5 10.1531 4.58258 10.7087 4 10.9147V13.5C4 14 4.5 14 4.5 14H7V15H4C4 15 3 15 3 14V10.9147C2.41742 10.7087 2 10.1531 2 9.5C2 8.67163 2.67157 8 3.5 8C4.32843 8 5 8.67163 5 9.5ZM4 3H7V1.5L10 3.5L7 5.5V4H4.5C4 4 4 4.5 4 4.5V7H3V4C3 4 3 3 4 3ZM11 12.5L8 14.5L11 16.5V15H14C15 15 15 14 15 14V11H14V13.5C14 13.5 14 14 13.5 14H11V12.5ZM14 7H13V10H16V7H15V4C15 3 14 3 14 3H11V4H13.5C13.5 4 14 4 14 4.5V7Z"
        fill="#283354"
      />{' '}
    </svg>
  );
}

export default IconAccessMethods;
