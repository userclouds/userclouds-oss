import { useState, useRef } from 'react';
import { connect } from 'react-redux';

import {
  useOutsideClickDetector,
  DropDown,
  DropDownOverflow,
} from '@userclouds/sharedui';

import { Button, IconButton, IconCopy } from '@userclouds/ui-component-lib';
import { RootState } from '../store';
import { GetAuthURL, GetLogoutURL } from '../Auth';
import Company from '../models/Company';
import Styles from './ProfileWidget.module.css';

type ProfileWidgetProps = {
  companies: Company[] | undefined;
  selectedCompanyID: string | undefined;
  displayName: string | undefined;
  email: string | undefined;
  userID: string | undefined;
  pictureURL: string | undefined;
  impersonatorName: string | undefined;
  impersonatorUserID: string | undefined;
  unimpersonateUser: () => void;
};
const ProfileWidget = ({
  companies,
  selectedCompanyID,
  displayName,
  email,
  userID,
  pictureURL,
  impersonatorName,
  impersonatorUserID,
  unimpersonateUser,
}: ProfileWidgetProps) => {
  const [dropDownActive, setDropDownActive] = useState(false);
  const toggleDropDown = () => {
    setDropDownActive(!dropDownActive);
  };

  const dropDownRef = useRef<HTMLDivElement>(null);
  useOutsideClickDetector([dropDownRef], () => {
    setDropDownActive(false);
  });

  const selectedCompany = (companies || []).find(
    (o) => o.id === selectedCompanyID
  );

  return (
    <div className={Styles.loginwidgetnavmenu} ref={dropDownRef}>
      {userID ? (
        <button onClick={toggleDropDown}>
          {pictureURL && (
            <img
              key="profileimg"
              className={Styles.profileimg}
              src={pictureURL}
              alt=""
            />
          )}
        </button>
      ) : (
        <a key="signinlink" href={GetAuthURL('/')}>
          <span className={Styles.profilename}>Hello! Sign In</span>
        </a>
      )}
      <DropDown
        active={dropDownActive}
        overflow={DropDownOverflow.Left}
        className={Styles.loginwidgetdropdown}
      >
        <dl className={Styles.loginwidgetcontent}>
          <dt>
            {displayName} ({email})
          </dt>
          <div>
            <dt>User ID:</dt>
            <dd>
              {userID}
              &nbsp;
              <IconButton
                icon={<IconCopy />}
                onClick={() => {
                  navigator.clipboard.writeText(userID || '');
                }}
                title="Copy user ID to clipboard"
                aria-label="Copy user ID to clipboard"
              />
            </dd>
          </div>
          {selectedCompany && (
            <div>
              <dt>Company:</dt>
              <dd>{selectedCompany.name}</dd>
            </div>
          )}
          {impersonatorUserID && (
            <button
              className={Styles.loginwidgetimpersonationbutton}
              onClick={unimpersonateUser}
            >
              Impersonator: {impersonatorName} ({impersonatorUserID})
            </button>
          )}
          <hr />
          <Button size="small" theme="ghost" href={GetLogoutURL('/')}>
            Logout
          </Button>
        </dl>
      </DropDown>
    </div>
  );
};

export default connect((state: RootState) => ({
  companies: state.companies,
  selectedCompanyID: state.selectedCompanyID,
}))(ProfileWidget);
