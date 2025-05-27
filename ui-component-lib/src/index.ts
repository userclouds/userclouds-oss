// eslint-disable-next-line import/order
import GlobalStyles from '../styles/globals-pages.css';
// eslint-disable-next-line import/order
import Variables from './_variables.module.css';
import { Accordion, AccordionItem } from '@components/Accordion';
import Button from '@components/Button';
import IconButton from '@components/IconButton';
import Anchor from '@components/Anchor';
import AppliedFilter from '@components/AppliedFilter';
import Breadcrumbs from '@components/Breadcrumbs';
import Checkbox from '@components/Checkbox';
import CodeEditor from '@components/CodeEditor';
import DataListTagger from '@components/DataListTagger';
import DateTimePicker from '@components/DateTimePicker';
import Dialog, { DialogBody, DialogFooter } from '@components/Dialog';
import FormNote from '@components/FormNote';
import Heading from '@components/Heading';
import HiddenTextInput from '@components/HiddenTextInput';
import InlineNotification from '@components/InlineNotification';
import Label from '@components/Label';
import LoaderDots from '@components/LoaderDots';
import { UserAvatar } from '@components/Avatar';
import Radio from '@components/Radio';
import Select, { PseudoSelect } from '@components/Select';
import TabGroup from '@components/TabGroup';
import Tag from '@components/Tag';
import Text, { ErrorText, SuccessText } from '@components/Text';
import TextArea from '@components/TextArea';
import TextInput from '@components/TextInput';
import Toast from '@components/Toast';
import ToastSolid from '@components/ToastSolid';
import ToolTip from '@components/Tooltip';
import InputReadOnly, { InputReadOnlyHidden } from '@components/InputReadOnly';
import {
  Table,
  TableHead,
  TableBody,
  TableFoot,
  TableRowHead,
  TableRow,
  TableCell,
  TableTitle,
} from '@components/Table';
import TextShortener from '@components/TextShortener';
import HorizontalRule from '@components/HorizontalRule';
import ButtonGroup from '@layouts/ButtonGroup';
import {
  Card,
  CardRow,
  CardColumns,
  CardColumn,
  CardFooter,
} from '@layouts/Card';
import EmptyState from '@layouts/EmptyState';
import { Dropdown, DropdownSection, DropdownButton } from '@layouts/Dropdown';

import {
  IconAccessMethods,
  IconAccessPermissions,
  IconAccessRules,
  IconAdd,
  IconArrowLeft,
  IconArrowDown,
  IconArrowRight,
  IconAuthenticatorApp,
  IconBarChartBox,
  IconBookOpen,
  IconCheck,
  IconCheckmark,
  IconClose,
  IconCopy,
  IconDashboard3,
  IconDash,
  IconDatabase2,
  IconDeleteBin,
  IconEdit,
  IconEmail,
  IconEye,
  IconEyeOff,
  IconFacebookBlackAndWhite,
  IconFileCode,
  IconFileList2,
  IconFilter,
  IconFocus3Line,
  IconGoogleBlackAndWhite,
  IconLinkedInBlackAndWhite,
  IconLock2,
  IconLogin,
  IconManageTeam,
  IconMenu,
  IconMenuCheck,
  IconMenuHome,
  IconMicrosoftInBlackAndWhite,
  IconMonitoring,
  IconNews,
  IconOrganization,
  IconRecoveryCode,
  IconRotate,
  IconSettingsGear,
  IconShieldKeyhole,
  IconSms,
  IconStarLine,
  IconStarSolid,
  IconTeam,
  IconTerminalBox,
  IconToggleOff,
  IconToggleOn,
  IconUser3,
  IconUserAuthentication,
  IconUserDataMapping,
  IconUserDataMasking,
  IconUserDataStorage,
  IconUserSearch,
  IconUserReceived2,
} from '@icons';
import SideBarStyles from '../styles/SideBar.module.scss';
import FooterStyles from '../styles/Footer.module.scss';
import HeaderStyles from '../styles/Header.module.scss';
import PageWrapStyles from '../styles/PageWrap.module.scss';
import PageTitleStyles from '../styles/PageTitle.module.scss';

export {
  Variables,
  GlobalStyles,
  PageTitleStyles,
  PageWrapStyles,
  HeaderStyles,
  FooterStyles,
  SideBarStyles,
  // basic components
  Anchor,
  Accordion,
  AccordionItem,
  UserAvatar,
  Button,
  ButtonGroup,
  Breadcrumbs,
  Checkbox,
  CodeEditor,
  DataListTagger,
  DateTimePicker,
  EmptyState,
  ErrorText,
  AppliedFilter,
  FormNote,
  Heading,
  HiddenTextInput,
  IconButton,
  InlineNotification,
  LoaderDots,
  Radio,
  Select,
  PseudoSelect,
  SuccessText,
  Text,
  Table,
  TableHead,
  TableBody,
  TableFoot,
  TableRowHead,
  TableRow,
  TableCell,
  TableTitle,
  Tag,
  Toast,
  ToastSolid,
  ToolTip,
  Label,
  TabGroup,
  TextArea,
  TextInput,
  InputReadOnly,
  InputReadOnlyHidden,
  TextShortener,
  HorizontalRule,
  // layout components
  Card,
  CardRow,
  CardColumns,
  CardColumn,
  CardFooter,
  Dialog,
  DialogBody,
  DialogFooter,
  Dropdown,
  DropdownSection,
  DropdownButton,
  // Icons
  IconAccessMethods,
  IconAccessPermissions,
  IconAccessRules,
  IconAdd,
  IconArrowLeft,
  IconArrowDown,
  IconArrowRight,
  IconAuthenticatorApp,
  IconBarChartBox,
  IconBookOpen,
  IconCheck,
  IconCheckmark,
  IconClose,
  IconCopy,
  IconDashboard3,
  IconDash,
  IconDatabase2,
  IconDeleteBin,
  IconEdit,
  IconEmail,
  IconEye,
  IconEyeOff,
  IconFacebookBlackAndWhite,
  IconFileCode,
  IconFileList2,
  IconFilter,
  IconFocus3Line,
  IconGoogleBlackAndWhite,
  IconLinkedInBlackAndWhite,
  IconLock2,
  IconLogin,
  IconManageTeam,
  IconMenu,
  IconMenuCheck,
  IconMenuHome,
  IconMicrosoftInBlackAndWhite,
  IconMonitoring,
  IconNews,
  IconOrganization,
  IconRecoveryCode,
  IconRotate,
  IconSettingsGear,
  IconShieldKeyhole,
  IconStarLine,
  IconStarSolid,
  IconSms,
  IconTeam,
  IconTerminalBox,
  IconToggleOff,
  IconToggleOn,
  IconUser3,
  IconUserAuthentication,
  IconUserDataMapping,
  IconUserDataMasking,
  IconUserDataStorage,
  IconUserSearch,
  IconUserReceived2,
};
