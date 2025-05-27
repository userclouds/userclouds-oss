import { IconTypes, getSize } from '../iconHelpers';

function IconFilter({ className, size = 'medium' }: IconTypes): JSX.Element {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={getSize(size)}
      height={getSize(size)}
      className={className}
      fill="#ffffff"
    >
      <title>Filter</title>
      <path
        d="M2 1H18C18.5523 1 19 1.44772 19 2V2.64316C19 2.84435 18.9393 3.04086 18.8259 3.20702L13.2886 11.3173C12.9483 11.8158 12.7662 12.4053 12.7662 13.0089V17.4853C12.7662 17.7755 12.6402 18.0514 12.4208 18.2413L8.88833 21.2998C8.24067 21.8605 7.23377 21.4005 7.23377 20.5438V13.0089C7.23377 12.4053 7.05171 11.8158 6.71138 11.3173L1.17413 3.20702C1.06069 3.04086 1 2.84435 1 2.64316V2C1 1.44771 1.44771 1 2 1Z"
        stroke="#283354"
        strokeWidth="2"
      />
    </svg>
  );
}

export default IconFilter;
