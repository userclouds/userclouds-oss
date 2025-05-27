import { connect } from 'react-redux';
import clsx from 'clsx';
import { AppDispatch } from '../store';
import { redirect } from '../routing';
import PageCommon from '../pages/PageCommon.module.css';

const linkHandler = (e: React.MouseEvent<HTMLAnchorElement>) => async () => {
  const { href } = e.currentTarget as HTMLAnchorElement;

  redirect(href);
};

interface LinkProps extends React.AnchorHTMLAttributes<HTMLAnchorElement> {
  applyStyles?: boolean;
  dispatch: AppDispatch;
}
const RouterLink = ({
  applyStyles = true,
  href,
  children,
  onClick,
  dispatch,
  ...otherProps
}: LinkProps) => {
  return (
    <a
      href={href}
      className={clsx(otherProps.className, applyStyles ? PageCommon.link : '')}
      {...otherProps}
      onClick={(e: React.MouseEvent<HTMLAnchorElement>) => {
        e.preventDefault();

        if (onClick) {
          onClick(e);
        }
        if (href) {
          dispatch(linkHandler(e));
        }
      }}
    >
      {children}
    </a>
  );
};

export default connect()(RouterLink);
