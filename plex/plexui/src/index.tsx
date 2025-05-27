import React from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Login from './pages/Login';
import PasswordlessLogin from './pages/PasswordlessLogin';
import MFAChannel from './pages/MFAChannel';
import MFARecoveryCode from './pages/MFARecoveryCode';
import MFASubmit from './pages/MFASubmit';
import NotFound from './pages/NotFound';
import CreateUser from './pages/CreateUser';
import StartResetPassword from './pages/StartResetPassword';
import FinishResetPassword from './pages/FinishResetPassword';
import EmailExistsPage from './pages/EmailExists';
import './index.module.css';
import OTPVerifyEmail from './pages/OTPVerifyEmail';
import MFAConfigure from './pages/MFAConfigure';
import MFAChannelSelector from './pages/MFAChannelSelector';
import MFAConfigureChannel from './pages/MFAConfigureChannel';

const container = document.getElementById('root');
if (!container) {
  throw new Error('Failed to find the root element');
}

const root = createRoot(container);
root.render(
  <React.StrictMode>
    <BrowserRouter basename="/plexui">
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/passwordlesslogin" element={<PasswordlessLogin />} />
        <Route path="/mfachannel" element={<MFAChannel />} />
        <Route path="/mfachannel/configure" element={<MFAConfigure />} />
        <Route
          path="/mfachannel/configurechannel"
          element={<MFAConfigureChannel />}
        />
        <Route path="/mfachannel/choose" element={<MFAChannelSelector />} />
        <Route path="/mfashowrecoverycode" element={<MFARecoveryCode />} />
        <Route path="/mfasubmit" element={<MFASubmit />} />
        <Route path="/createuser" element={<CreateUser />} />
        <Route path="/startresetpassword" element={<StartResetPassword />} />
        <Route path="/finishresetpassword" element={<FinishResetPassword />} />
        <Route path="/userwithemailexists" element={<EmailExistsPage />} />
        <Route path="/otp/submit" element={<OTPVerifyEmail />} />
        <Route path="*" element={<NotFound />} />
      </Routes>
    </BrowserRouter>
  </React.StrictMode>
);
