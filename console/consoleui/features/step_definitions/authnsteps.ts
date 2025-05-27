import { Given, When, Then } from '@cucumber/cucumber';
import { expect } from '@playwright/test';

import TenantPlexConfig from '../../src/models/TenantPlexConfig';
import LoginApp from '../../src/models/LoginApp';
import { CukeWorld } from '../support/world';
import { DEBUG_MODE, PORT, HOST } from '../support/globalSetup';
import tenantsMock from '../fixtures/tenants.json';
import plexConfigMock from '../fixtures/plexconfig.json';
import emailElementsMock from '../fixtures/email_elements.json';
import smsElementsMock from '../fixtures/sms_elements.json';
import modifiedPageParametersMock from '../fixtures/modified_pageparameters.json';
import { hexToRgb } from './helpers';

const editedApp: LoginApp = {
  id: '90ffb499-2549-470e-99cd-77f7008e2735',
  name: 'Foo',
  description: 'Foo',
  organization_id: '1ee4497e-c326-4068-94ed-3dcdaaaa53bc',
  client_id: 'ed79c6c4d8ee35cb50f3e5ed6d788509',
  client_secret:
    '4DzhbWxUu4X/igtUZ0pvt/vC+lQM9nOaXOmBdyn4pxksNFRxYuD/yFLF6NMoBfTf',
  restricted_access: false,
  token_validity: {
    access: 86400,
    refresh: 2592000,
    impersonate_user: 3600,
  },
  provider_app_ids: ['3e3de5b2-f789-412b-8df9-859b73acbb98'],
  allowed_redirect_uris: [
    'https://foo.com',
    'https://console.dev.userclouds.tools:3010/auth/callback',
    'https://console.dev.userclouds.tools:3010/auth/invitecallback',
  ],
  allowed_logout_uris: ['https://foo.com'],
  message_elements: {},
  page_parameters: {},
  grant_types: ['authorization_code', 'refresh_token'],
  synced_from_provider: '00000000-0000-0000-0000-000000000000',
  impersonate_user_config: {
    check_attribute: '',
    bypass_company_admin_check: false,
  },
  saml_idp: {
    certificate:
      '-----BEGIN CERTIFICATE-----\nMIICjTCCAXWgAwIBAgIRAmSPqwX5h9//1Dzduk+aWMMwDQYJKoZIhvcNAQELBQAw\nADAiGA8wMDAxMDEwMTAwMDAwMFoYDzAwMDEwMTAxMDAwMDAwWjAAMIIBIjANBgkq\nhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA1702uWbq1eZcb3m+Jej/S9DsVq5pkrUq\nsw36V+Z0wC0sPhMdnMB09NlzJF9zxFasu/XA87z7UxYvjiwjeTKKlm3vywM7wkyR\n4KdZ3z51kUIb0F32rJfxnthXp8dwn4fSV0Bgdrog+dMSDpcPDRTV+/NdHWKFM8qf\n928skv7C5zIfd+DEzNI29SqZF7oX+pXrKS0G9BNUaFl3hUDKGG38TLBUXdaXZ7F6\nsRf+W95F2BYEcpmRUB6EIDn6gH97Vb0svdsO2Ef+7fLhCG6EKKo2uLMr/M+KF5Uy\ny0EaSCJjFmoMPkhhaKTp1cPHtEQwPlsaDR2XwZLH+csW9DXUWnkDqwIDAQABMA0G\nCSqGSIb3DQEBCwUAA4IBAQDIuvxzkoPzrQS6nzcrrB6t68Ky52981V4shD3c+jCX\nTtdf3jp0NLnl5qkcC2/xy/tHhXJXH1NDpw780z5YG81VCQZ+jSkIcNA+QLIY4aCo\nWl9I3oRfmyGrvlyIjNcBRdddjwmlK6gq91IoXAKzAYc2oO/ECpwSSb+Wve8F/1ZZ\ncHXzt7a3VlJ/x99xB9fwg+G+1KkiMUoBpddraMWmES1aEF33cmjipErWPu6I3634\nqdnd6GYNeyfWDRZlqGWFxI7mHIH8P5hEduTIDZR9WCAVlwkMF+5YJhhk4B5xzGJE\nG5N2N9MCQqtYCjHMq3lyy/HpxihY7fvNk/E1sobnwz4N\n-----END CERTIFICATE-----\n',
    private_key:
      'dev://LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcEFJQkFBS0NBUUVBMTcwMnVXYnExZVpjYjNtK0plai9TOURzVnE1cGtyVXFzdzM2VitaMHdDMHNQaE1kCm5NQjA5Tmx6SkY5enhGYXN1L1hBODd6N1V4WXZqaXdqZVRLS2xtM3Z5d003d2t5UjRLZFozejUxa1VJYjBGMzIKckpmeG50aFhwOGR3bjRmU1YwQmdkcm9nK2RNU0RwY1BEUlRWKy9OZEhXS0ZNOHFmOTI4c2t2N0M1eklmZCtERQp6TkkyOVNxWkY3b1grcFhyS1MwRzlCTlVhRmwzaFVES0dHMzhUTEJVWGRhWFo3RjZzUmYrVzk1RjJCWUVjcG1SClVCNkVJRG42Z0g5N1ZiMHN2ZHNPMkVmKzdmTGhDRzZFS0tvMnVMTXIvTStLRjVVeXkwRWFTQ0pqRm1vTVBraGgKYUtUcDFjUEh0RVF3UGxzYURSMlh3WkxIK2NzVzlEWFVXbmtEcXdJREFRQUJBb0lCQUgreXVCbTJHWHJUQ0JQVAo5QUZza1BESGtaMGRUOUJPL0I5UVBzYVkycktHQ3BJVHJvdUNQN2hPbmlFQmZ2elFjUjR3c1MyVXh3Ni9LeGIwCmVXcmJ4N3lUQmtVY2ZOcmRoOXQ2TTNBNUFFNGkyMlBTdXBnZXVCNVY0RXluZUxwMUlzUVNqd2EzMVowS21yMlkKSElpWnRLK1Z0YUFYR05FM05zaTNQYU1rdVNrdkl2WnZOMVM1V002NkVBZHFiZktCY2RxTllQV0FOZlpBY212UAoyTWp0eEFNU3c1ZlNwQi9vVUdUZ0xsYmMrQkpBSEdTcVN1eDdCdzVCYlY4NUdpaW5vdWlOdHc4dzFGTTVEU0VPCnhSbGN3b0pIN1NPa1BrVE5kK1R2RHR6NGpINmFlaXV5bG5BZ2FwL240WndQaWRJbDBNOVUyWFJadHBraW9yWXUKQXROUGVVRUNnWUVBL2VBWFUyaXkydjRyUXVCdlZPN3hRSUhxS3hESkZ4WjY5TUlETnFwdTJTSGJHRjlMYVVhcgpTZDlGT1VmVUM2ZU50STIwdVBQYURUN25PdkxmNEhHeEJ0aEU2ZEJTbkZ6bTVjaWc4amV3ZHN2Q3pSM011Y0lnCnpNL2RQNDAxRUYzZS9KemE5QWNkZEdjWGlOM3FUaHhOYU5kQ2hPYnYyL21IMnZ4cGdtZWo2S2NDZ1lFQTJZdHIKS3Jub3JqZlBXdG9sZ2FkSS9KM2VhN3ZLeFdHUFF4Q0VQTHgxYXNBL3lFQisybUZaNjYxeHRtaGlNSllPREpwQQpWeTQraHk5bko5N2lVL254L2NXV3g5NlJoeExWTGJUdGsxMnF2bWFkSzZNVWlrVE12c1FzamJxNFFVM253MDFMCmFlRHQ5NThFRmRmRTI5YmdiWXY2bGpMVnZsQ3BESnNqb3FtWGFWMENnWUVBMUYwd2hlZ052TnhpQ2NZOXV0bEoKVzRHUkJVYzhQeURoNTMyblBJSWl5V1RscGlTSXExNmZCK05KUDVvVENWQzJXN014Mm9pNC9OMkNoVEFIRC9OcQpkdVJQK1JuM0VLOHh3a01xUnBOSS9JYUR4QnJLVnhUSlpTbjMxQ0psb2ZRMEJER2RnZ1cxb05wZnVIQ1JmNWR6Ck5XRGpWdExyRDZKUy8xNm5UNXNzWS84Q2dZQkYyTisxdmk4WkVNNUF5MTNUZlJTUUYxZjhtelVGbnNkU3J4RG0KTjFRenpEb3VYNWJiSXZxdUV1ZzV1dFliNTNIblZmZG1obkNKRXcwNTNmUXBKazB1UDZ5ank3QktBQi8ySnV0SQpyNEJNMWNHTTZ6V0RGNGZ0a0NzRjduZU9jQ2NEcStPVXdTVm1wZVczNWFsTk5IYW1kWlVsZUhqc1BCV3ErSHkrCmsxa0wrUUtCZ1FEVkg4UUZNMVdxU2w3eCtRbk5hdlNabmFvcVBJK1FFNXFoaXdCTlJiSUs2TDh2UExnQkhpWFoKdlcvdGY2d3Nzbjh6UENYVDZoL0xQQmdEMmdYZVB6TXBrSWJnNXpmMGZPejRsSHBNazZ4emp6dUhUTnRBUGNDWAo1SllwVmtkWGxNWkpvMHlxSGFRY2hFK2lrTFFjQytnQUgwa29lcVdmZTFNZ2NnT05DQVBmWnc9PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=',
    metadata_url:
      'https://usercloudsdev-august172023.tenant.dev.userclouds.tools:3333/saml/metadata/ed79c6c4d8ee35cb50f3e5ed6d788509',
    sso_url:
      'https://usercloudsdev-august172023.tenant.dev.userclouds.tools:3333/saml/sso/ed79c6c4d8ee35cb50f3e5ed6d788509',
  },
};
const additionalApp = {
  allowed_logout_uris: [],
  allowed_redirect_uris: [],
  client_id: '4cbdc4becb3324243c0cd81613ecb714',
  client_secret:
    'IlYUNTl1uQuuVwo7UG8i2V8Q1qjXRtMDZ4y3kOtStoy4vdBYdGqxScO+mRuVJNHA',
  description: '',
  grant_types: ['authorization_code', 'refresh_token', 'client_credentials'],
  id: '61dc325e-b741-4f5e-8f54-b4fdac202b85',
  impersonate_user_config: {
    check_attribute: '',
    bypass_company_admin_check: false,
  },
  message_elements: {},
  name: 'New Plex App',
  organization_id: '00000000-0000-0000-0000-000000000000',
  page_parameters: {},
  provider_app_ids: [],
  restricted_access: false,
  synced_from_provider: '00000000-0000-0000-0000-000000000000',
  token_validity: {
    access: 86400,
    refresh: 2592000,
    impersonate_user: 3600,
  },
};
Given('a modified login app', async function (this: CukeWorld) {
  await this.page.route(
    `${HOST}:${PORT}/api/tenants/${tenantsMock[0].id}/plexconfig`,
    (route) => {
      if (route.request().method() === 'POST') {
        const responseBody: TenantPlexConfig = JSON.parse(
          JSON.stringify(plexConfigMock as unknown as TenantPlexConfig)
        );
        responseBody.tenant_config.plex_map.apps =
          plexConfigMock.tenant_config.plex_map.apps.filter(
            (app: LoginApp) => app.id !== editedApp.id
          );
        responseBody.tenant_config.plex_map.apps.push(editedApp);
        const response = {
          status: 200,
          body: JSON.stringify(responseBody),
        };
        if (DEBUG_MODE) {
          console.log(
            `Preparing to respond to POST request with status 200`,
            response.body
          );
        }
        // introduce a slight delay to simulate real conditions
        setTimeout(() => {
          route.fulfill(response);
        }, 200);
      }
    },
    { times: 1 }
  );
});

Given(
  'modified login settings for a login app',
  async function (this: CukeWorld) {
    await this.page.route(
      `${HOST}:${PORT}/api/tenants/${tenantsMock[0].id}/apppageparameters/90ffb499-2549-470e-99cd-77f7008e2735`,
      (route) => {
        if (route.request().method() === 'PUT') {
          const response = {
            status: 200,
            body: JSON.stringify(modifiedPageParametersMock),
          };
          if (DEBUG_MODE) {
            console.log(
              `Preparing to respond to PUT request with status 200`,
              response.body
            );
          }
          // introduce a slight delay to simulate real conditions
          setTimeout(() => {
            route.fulfill(response);
          }, 200);
        }
      },
      { times: 1 }
    );
  }
);

Given(
  'modified email settings for a login app',
  async function (this: CukeWorld) {
    await this.page.route(
      `${HOST}:${PORT}/api/tenants/${tenantsMock[0].id}/emailelements`,
      (route) => {
        if (route.request().method() === 'POST') {
          const saveEmailElementsResponse = JSON.parse(
            JSON.stringify(emailElementsMock)
          );
          saveEmailElementsResponse.tenant_app_message_elements.app_message_elements[0].message_type_message_elements.passwordless_login.message_elements.subject_template.custom_value =
            'Login Request from {{.AppName}}';
          saveEmailElementsResponse.tenant_app_message_elements.app_message_elements[0].message_type_message_elements.passwordless_login.message_elements.text_template.custom_value =
            '{{.AppName}}: confirm your email address to sign in.';
          const response = {
            status: 200,
            body: JSON.stringify(saveEmailElementsResponse),
          };
          if (DEBUG_MODE) {
            console.log(
              `Preparing to respond to POST request with status 200`,
              response.body
            );
          }
          // introduce a slight delay to simulate real conditions
          setTimeout(() => {
            route.fulfill(response);
          }, 200);
        }
      },
      { times: 1 }
    );
  }
);

Given(
  'modified SMS settings for a login app',
  async function (this: CukeWorld) {
    await this.page.route(
      `${HOST}:${PORT}/api/tenants/${tenantsMock[0].id}/smselements`,
      (route) => {
        if (route.request().method() === 'POST') {
          const saveSMSElementsResponse = JSON.parse(
            JSON.stringify(smsElementsMock)
          );
          saveSMSElementsResponse.tenant_app_message_elements.app_message_elements[0].message_type_message_elements.sms_mfa_verify.message_elements.sms_sender.custom_value =
            '+15555552343';
          saveSMSElementsResponse.tenant_app_message_elements.app_message_elements[0].message_type_message_elements.sms_mfa_verify.message_elements.sms_body_template.custom_value =
            'Verify your phone number with {{.AppName}}. Use code {{.Code}}';
          const response = {
            status: 200,
            body: JSON.stringify(saveSMSElementsResponse),
          };
          if (DEBUG_MODE) {
            console.log(
              `Preparing to respond to POST request with status 200`,
              response.body
            );
          }
          // introduce a slight delay to simulate real conditions
          setTimeout(() => {
            route.fulfill(response);
          }, 200);
        }
      },
      { times: 1 }
    );
  }
);

Given('an additional login app', async function (this: CukeWorld) {
  await this.page.route(
    `${HOST}:${PORT}/api/tenants/${tenantsMock[0].id}/loginapps`,
    (route) => {
      if (route.request().method() === 'POST') {
        const responseBody: TenantPlexConfig = JSON.parse(
          JSON.stringify(plexConfigMock as unknown as TenantPlexConfig)
        );
        responseBody.tenant_config.plex_map.apps.push(additionalApp);
        const response = {
          status: 200,
          body: JSON.stringify(responseBody.tenant_config),
        };
        if (DEBUG_MODE) {
          console.log(
            `Preparing to respond to POST request with status 200`,
            response.body
          );
        }
        // introduce a slight delay to simulate real conditions
        setTimeout(() => {
          route.fulfill(response);
        }, 200);
      }
    },
    { times: 1 }
  );
});

When(
  /I click the "([A-Za-z]+)" button associated with the "([A-Za-z\-_0-9]+)" input in the (email settings|SMS settings) form/,
  async function (
    this: CukeWorld,
    buttonLabel: string,
    inputName: string,
    formType: string
  ) {
    const formID =
      formType === 'email settings' ? 'emailSettings' : 'smsSettings';
    const form = this.page.locator(`form#${formID}`);
    const fieldset = form.locator('fieldset', {
      has: this.page.locator(`[name="${inputName}"]`),
    });
    const paramChooser = fieldset.locator('ul');
    const button = paramChooser.locator(`button:has-text("${buttonLabel}")`);
    await button.click();
  }
);

Then(
  'I should see the following text in the login preview pane',
  async function (this: CukeWorld, testData) {
    const pane = this.page.locator('#loginPagePreview');
    for (const [selector, textContent] of testData.rows()) {
      const el = await pane
        .locator(`${selector}:has-text("${textContent}")`)
        .first();
      expect(await el.textContent()).toContain(textContent);
    }
  }
);

Then(
  /social login should (precede|follow) other methods in the login form/,
  async function (this: CukeWorld, precedeOrFollow) {
    const precede = precedeOrFollow === 'precede';
    const loginForm = this.page.locator('#loginPagePreview #loginForm');
    const children = await loginForm.evaluate((el) => {
      const list: any = [];
      const kids = el.childNodes;
      kids.forEach((child) => {
        list.push({
          nodeType: child.nodeType,
          nodeName: child.nodeName,
          className: child.nodeType === 1 ? (child as Element).className : '',
        });
      });
      return list;
    });
    let encounteredOther = false;
    for (let i = 0; i < children.length; i++) {
      const child = children[i];
      if (child.nodeName === 'UL' && child.className.includes('socialLogin')) {
        break;
      } else if (child.nodeName === 'FIELDSET') {
        encounteredOther = true;
      }
    }
    expect(encounteredOther).toBe(!precede);
  }
);

Then(
  'the submit button in the login preview pane should have a {string} of {string}',
  async function (this: CukeWorld, cssProperty: string, buttonColor: string) {
    const button = this.page.locator(
      '#loginPagePreview form button[type="submit"]'
    );
    const color = await button.evaluate((element, cssProp) => {
      return window.getComputedStyle(element).getPropertyValue(cssProp);
    }, cssProperty);
    expect(color).toEqual(hexToRgb(buttonColor));
  }
);
