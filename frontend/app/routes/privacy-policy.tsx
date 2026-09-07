import { Link } from "react-router";
import { LegalPage, LEGAL_CONTACT_EMAIL } from "~/components/LegalPage";
import { pageMeta } from "~/lib/pageTitle";

export function meta() {
  return pageMeta(
    "Privacy Policy",
    "How Glossias collects, uses, and shares personal information",
  );
}

export default function PrivacyPolicy() {
  return (
    <LegalPage title="Privacy Policy">
      <p>
        Titus Murphy, operating Glossias (&quot;<strong>Glossias</strong>,&quot;
        &quot;<strong>we</strong>,&quot; &quot;<strong>us</strong>,&quot; or
        &quot;<strong>our</strong>&quot;), provides an interactive, story-based
        language-learning service for students in introductory language courses
        and their instructors. This Privacy Policy describes how Glossias
        processes personal information that we collect through our digital
        properties that link to this Privacy Policy, including{" "}
        <a href="https://glossias.org">glossias.org</a> (collectively, the
        &quot;<strong>Service</strong>&quot;).
      </p>
      <p>
        <strong>
          California Notice at Collection / State Privacy Rights Notice:
        </strong>{" "}
        See the <a href="#state-privacy-rights">State privacy rights notice</a>{" "}
        section below for important information about your rights under
        applicable state privacy laws.
      </p>
      <p>
        Related terms for using the Service are in our{" "}
        <Link to="/terms-of-service">Terms of Service</Link>.
      </p>

      <h2>Index</h2>
      <ul>
        <li>
          <a href="#personal-information">Personal information we collect</a>
        </li>
        <li>
          <a href="#tracking">Tracking and other technologies</a>
        </li>
        <li>
          <a href="#how-we-use">How we use your personal information</a>
        </li>
        <li>
          <a href="#retention">Retention</a>
        </li>
        <li>
          <a href="#how-we-share">How we share your personal information</a>
        </li>
        <li>
          <a href="#your-choices">Your choices</a>
        </li>
        <li>
          <a href="#other-sites">Other sites and services</a>
        </li>
        <li>
          <a href="#security">Security</a>
        </li>
        <li>
          <a href="#international">International data transfer</a>
        </li>
        <li>
          <a href="#children">Children</a>
        </li>
        <li>
          <a href="#changes">Changes to this Privacy Policy</a>
        </li>
        <li>
          <a href="#contact">How to contact us</a>
        </li>
        <li>
          <a href="#state-privacy-rights">State privacy rights notice</a>
        </li>
      </ul>

      <h2 id="personal-information">Personal information we collect</h2>
      <h3>Information you provide to us</h3>
      <p>
        Personal information you may provide to us through the Service or
        otherwise includes:
      </p>
      <ul>
        <li>
          <strong>Contact data</strong>, such as your name and email address.
        </li>
        <li>
          <strong>Profile and account data</strong>, such as the credentials and
          profile details you set with our authentication provider when you
          create an account, and the role associated with your account (student,
          course instructor/administrator, or super administrator).
        </li>
        <li>
          <strong>Course data</strong>, such as the courses you are enrolled in
          or administer.
        </li>
        <li>
          <strong>Learning activity and user-generated content</strong>, such as
          exercise answers (including identification choices, translation
          requests, written Produce submissions, and recall orderings), scores,
          feedback shown to you, and time spent on story phases.
        </li>
        <li>
          <strong>Communications data</strong> based on our exchanges with you,
          including when you email us or report a problem.
        </li>
        <li>
          <strong>Other data</strong> not specifically listed here, which we
          will use as described in this Privacy Policy or as otherwise disclosed
          at the time of collection.
        </li>
      </ul>
      <p>
        We do not ask you for payment card numbers, government-issued
        identification numbers, or similar financial or identity documents. The
        Service is not a paid storefront.
      </p>

      <h3>Third-party sources</h3>
      <p>
        We may combine personal information we receive from you with personal
        information falling within one of the categories identified above that
        we obtain from other sources, such as:
      </p>
      <ul>
        <li>
          <strong>Authentication providers.</strong> We use{" "}
          <a href="https://clerk.com">Clerk</a> to create and sign in to
          accounts. Clerk may provide us with your user identifier, name, email
          address, and related account metadata. Clerk processes that
          information under{" "}
          <a href="https://clerk.com/legal/privacy">its privacy policy</a>.
        </li>
        <li>
          <strong>Instructors and course administrators.</strong> An instructor
          may enroll you in a course by email address, or designate you as a
          course administrator.
        </li>
        <li>
          <strong>Service providers</strong> that provide services on our behalf
          or help us operate the Service, as described below.
        </li>
      </ul>

      <h3>Automatic data collection</h3>
      <p>
        We, our service providers, and our authentication provider may
        automatically log information about you, your computer or mobile device,
        and your interaction over time with the Service, such as:
      </p>
      <ul>
        <li>
          <strong>Device data</strong>, such as browser type, operating system,
          language settings, and IP address.
        </li>
        <li>
          <strong>Online activity data</strong>, such as pages or screens you
          viewed, navigation within a story, access times, and time spent on a
          story phase.
        </li>
      </ul>
      <p>
        We do not collect precise geolocation. We do not use advertising pixels
        or interest-based advertising partners.
      </p>
      <p>
        For more information concerning our automatic collection of data, please
        see the Tracking and other technologies section below.
      </p>

      <h2 id="tracking">Tracking and other technologies</h2>
      <h3>Cookies and similar technologies</h3>
      <p>
        Some of our automatic data collection is facilitated by cookies and
        similar technologies. We use these technologies as follows:
      </p>
      <ul>
        <li>
          <strong>Essential authentication cookies</strong> set by Clerk so we
          can keep you signed in and protect your account. The Service cannot
          function without these.
        </li>
        <li>
          <strong>Local device storage</strong> in your browser (for example, to
          keep a draft of written work if you reload a page). This stays on your
          device unless you clear site data.
        </li>
      </ul>
      <p>
        We do not use third-party advertising cookies, analytics pixels, or
        social-media tracking pixels on the Service. We do not maintain a
        separate Cookie Notice; this section is the notice for those
        technologies.
      </p>
      <p>
        You can typically remove or reject cookies through your browser
        settings. If you block essential cookies, you may not be able to sign in
        or use account features.
      </p>

      <h3>Fonts and other third-party resources</h3>
      <p>
        The Service loads fonts and icons from Google Fonts, which may receive
        your IP address and browser information. Google&apos;s use of that
        information is described in{" "}
        <a href="https://policies.google.com/privacy">
          Google&apos;s privacy policy
        </a>
        .
      </p>

      <h3>Artificial intelligence technologies</h3>
      <p>
        When you submit written work in the Produce phase, we may send your
        attempt, the corresponding story text, and related grading instructions
        to Anthropic so that an AI model can return a score and a short feedback
        sentence. We store that exchange so we can show you the result, let your
        instructors review it, and audit grading.
      </p>
      <p>
        We do not use your personal information to train our own machine
        learning models. Anthropic&apos;s handling of information sent through
        its API is governed by{" "}
        <a href="https://www.anthropic.com/privacy">
          Anthropic&apos;s privacy policy
        </a>{" "}
        and its API terms.
      </p>

      <h2 id="how-we-use">How we use your personal information</h2>
      <p>
        We may use your personal information for the following purposes or as
        otherwise described at the time of collection:
      </p>

      <h3>Service delivery and operations</h3>
      <ul>
        <li>provide the Service, including story exercises and scores;</li>
        <li>enable security features of the Service;</li>
        <li>establish and maintain your user profile;</li>
        <li>enroll you in courses and show instructors your progress;</li>
        <li>
          communicate with you about the Service, including Service-related
          announcements, updates, security alerts, and support messages; and
        </li>
        <li>
          provide support for the Service, and respond to your requests,
          questions, and feedback.
        </li>
      </ul>

      <h3>Service personalization</h3>
      <p>We may use your personal information to:</p>
      <ul>
        <li>remember your place in a story and any in-progress drafts; and</li>
        <li>
          personalize your experience with the Service (for example, showing the
          stories for your enrolled courses).
        </li>
      </ul>

      <h3>Service improvement</h3>
      <p>
        We may use your personal information to analyze usage of the Service,
        improve the Service, understand which story phases students use, and
        develop new features. We do not use Google Analytics or similar
        third-party advertising analytics.
      </p>

      <h3>No marketing or interest-based advertising</h3>
      <p>
        We do not send promotional marketing emails, and we do not use your
        personal information for interest-based advertising.
      </p>

      <h3>Compliance and protection</h3>
      <p>We may use your personal information to:</p>
      <ul>
        <li>
          comply with applicable laws, lawful requests, and legal process;
        </li>
        <li>
          protect our, your, or others&apos; rights, privacy, safety, or
          property (including by making and defending legal claims);
        </li>
        <li>
          audit our internal processes for compliance with legal and contractual
          requirements or our internal policies;
        </li>
        <li>enforce the terms that govern the Service; and</li>
        <li>
          prevent, identify, investigate, and deter fraudulent, harmful,
          unauthorized, unethical, or illegal activity, including cyberattacks.
        </li>
      </ul>

      <h3>Aggregated or de-identified data</h3>
      <p>
        We may create aggregated, de-identified, and/or anonymized data from
        personal information. We make personal information into de-identified
        and/or anonymized data by removing information that makes the data
        identifiable to you, and we will not attempt to reidentify any such data
        except to test whether our de-identification process complies with
        applicable law. We may use this data to understand how the Service is
        used and to improve it. We do not use it to train our own AI models.
      </p>

      <h2 id="retention">Retention</h2>
      <p>
        We generally retain personal information to fulfill the purposes for
        which we collected it, including operating the Service for your course,
        satisfying any legal, accounting, or reporting requirements,
        establishing or defending legal claims, or for fraud prevention. To
        determine the appropriate retention period, we may consider factors such
        as the amount, nature, and sensitivity of the personal information, the
        potential risk of harm from unauthorized use or disclosure, the purposes
        for which we process it and whether we can achieve those purposes
        through other means, and the applicable legal requirements.
      </p>
      <p>
        Course instructors may need access to scores and submissions for as long
        as they administer the relevant course. When we no longer require the
        personal information we have collected about you, we may either delete
        it, anonymize it, or isolate it from further processing.
      </p>

      <h2 id="how-we-share">How we share your personal information</h2>
      <p>
        We may share your personal information with the following parties (or as
        otherwise described in this Privacy Policy, in other applicable notices,
        or at the time of collection).
      </p>
      <ul>
        <li>
          <strong>Instructors and course administrators.</strong> If you are
          enrolled in a course, instructors and administrators of that course
          can view your enrollment, exercise answers, scores, time on task, and
          related progress. Super administrators of the Service can access this
          information to operate the Service.
        </li>
        <li>
          <strong>Service providers.</strong> Third parties that provide
          services on our behalf or help us operate the Service, including:
          <ul>
            <li>
              Clerk, for authentication (
              <a href="https://clerk.com/legal/privacy">privacy policy</a>);
            </li>
            <li>
              Supabase, for database hosting and file storage (
              <a href="https://supabase.com/privacy">privacy policy</a>);
            </li>
            <li>
              Anthropic, for AI scoring of Produce submissions (
              <a href="https://www.anthropic.com/privacy">privacy policy</a>);
              and
            </li>
            <li>
              Google, when the Service loads fonts and icons (
              <a href="https://policies.google.com/privacy">privacy policy</a>).
            </li>
          </ul>
        </li>
        <li>
          <strong>Professional advisors</strong>, such as lawyers or auditors,
          in the course of the professional services they render to us.
        </li>
        <li>
          <strong>Authorities and others.</strong> Law enforcement, government
          authorities, and private parties, as we believe in good faith to be
          necessary or appropriate for the compliance and protection purposes
          described above.
        </li>
        <li>
          <strong>Business transferees.</strong> We may disclose personal
          information in the context of actual or prospective business
          transactions (for example, a transfer of the Service). For example, we
          may need to share certain personal information with prospective
          counterparties and their advisers, or with a successor operator of the
          Service.
        </li>
      </ul>
      <p>
        We do not sell your personal information, and we do not share it with
        advertising partners for cross-context behavioral advertising. Your
        exercise answers are not posted publicly.
      </p>

      <h3>Education records</h3>
      <p>
        Glossias is designed to be used as part of a language course. If you use
        the Service through a school or instructor, your learning activity may
        be an education record of that course. We share that information with
        the instructors and administrators of courses you are enrolled in so
        they can teach and evaluate the course. We do not sell student
        information. If your school has a separate written agreement with us,
        that agreement controls to the extent of any conflict with this Privacy
        Policy.
      </p>

      <h2 id="your-choices">Your choices</h2>
      <p>
        In this section, we describe the rights and choices available to all
        users. Users who are located in certain U.S. states can find additional
        information about their rights below.
      </p>
      <ul>
        <li>
          <strong>Access or update your information.</strong> If you have
          registered for an account, you may review and update certain account
          information through the account menu on the Service (powered by
          Clerk).
        </li>
        <li>
          <strong>Cookies and local storage.</strong> You can control cookies
          and site data through your browser settings, as described above.
        </li>
        <li>
          <strong>Do Not Track.</strong> Some Internet browsers may be
          configured to send &quot;Do Not Track&quot; signals. We currently do
          not respond to those signals. We also do not use third-party
          advertising trackers on the Service. To learn more about Do Not Track,
          visit <a href="https://allaboutdnt.com">allaboutdnt.com</a>.
        </li>
        <li>
          <strong>Declining to provide information.</strong> We need to collect
          personal information to provide certain services. If you do not
          provide information we identify as required, we may not be able to
          provide those services (including an account).
        </li>
        <li>
          <strong>Close your account or delete your content.</strong> You may
          request that we close your account and delete personal information we
          hold by emailing{" "}
          <a href={`mailto:${LEGAL_CONTACT_EMAIL}`}>{LEGAL_CONTACT_EMAIL}</a>.
          We may retain information as required by law or as needed for
          instructors to maintain course records.
        </li>
      </ul>

      <h2 id="other-sites">Other sites and services</h2>
      <p>
        The Service may contain links to websites and other online services
        operated by third parties (for example, our status page, GitHub issue
        tracker, or embedded story videos). These links are not an endorsement
        of, or representation that we are affiliated with, any third party. We
        do not control websites or services operated by third parties, and we
        are not responsible for their actions. We encourage you to read the
        privacy policies of the other services you use.
      </p>

      <h2 id="security">Security</h2>
      <p>
        We employ technical, organizational, and physical safeguards designed to
        protect the personal information we collect, including authenticated
        access to the Service and encrypted transport. However, security risk is
        inherent in all internet and information technologies, and we cannot
        guarantee the security of your personal information.
      </p>
      <p>
        To report a security vulnerability, email{" "}
        <a href="mailto:security@glossias.org">security@glossias.org</a>.
      </p>

      <h2 id="international">International data transfer</h2>
      <p>
        We are based in the United States and use service providers that may
        operate in the United States or other countries. Your personal
        information may be transferred to the United States or other locations
        where privacy laws may not be as protective as those in your state,
        province, or country.
      </p>

      <h2 id="children">Children</h2>
      <p>
        The Service is not intended for use by anyone under 13 years of age, and
        you must be at least 18 years old to use the Service. If you are a
        parent or guardian of a child from whom you believe we have collected
        personal information in a manner prohibited by law, please contact us.
        If we learn that we have collected personal information through the
        Service from a child under 13 without the consent of the child&apos;s
        parent or guardian as required by law, we will comply with applicable
        legal requirements to delete the information.
      </p>

      <h2 id="changes">Changes to this Privacy Policy</h2>
      <p>
        We reserve the right to modify this Privacy Policy at any time. If we
        make material changes, we will notify you by updating the date of this
        Privacy Policy and posting it on the Service or other appropriate means.
        Any modifications will be effective upon our posting the modified
        version (or as otherwise indicated at the time of posting). In all
        cases, your use of the Service after the effective date of any modified
        Privacy Policy indicates your acknowledging that the modified Privacy
        Policy applies to your interactions with the Service.
      </p>

      <h2 id="contact">How to contact us</h2>
      <p>
        If you have questions about our practices or if you would like to
        exercise any privacy-related right that may be available to you, please
        email{" "}
        <a href={`mailto:${LEGAL_CONTACT_EMAIL}`}>{LEGAL_CONTACT_EMAIL}</a>.
      </p>

      <h2 id="state-privacy-rights">State privacy rights notice</h2>
      <p>
        Except as otherwise provided, this section applies to residents of U.S.
        states to the extent they have privacy laws applicable to us that grant
        their residents the rights described below (collectively the &quot;
        <strong>State Privacy Laws</strong>&quot;).
      </p>
      <p>
        This section describes how we collect, use, and share Personal
        Information of residents of these states and the rights these users may
        have. Please note that not all rights listed below may be afforded to
        all users, and that if you are not a resident of one of these states,
        you may not be able to exercise these rights. We may not be able to
        process your request if you do not provide us with sufficient detail to
        allow us to confirm your identity or understand and respond to it. We
        verify requests by matching information you provide (such as the email
        address on your account) to our records.
      </p>
      <p>
        For purposes of this section, the term &quot;
        <strong>Personal Information</strong>&quot; has the meaning given to
        &quot;personal data,&quot; &quot;personal information,&quot; or other
        similar terms, and &quot;<strong>Sensitive Personal Information</strong>
        &quot; has the meaning given to &quot;sensitive personal
        information,&quot; &quot;sensitive data,&quot; or other similar terms in
        the State Privacy Laws, except that in neither case does such term
        include information exempted from the scope of the State Privacy Laws.
      </p>

      <h3>Your privacy rights</h3>
      <p>
        The State Privacy Laws may provide residents with some or all of the
        rights listed below. These rights are not absolute, and some State
        Privacy Laws do not provide these rights to their residents. Therefore,
        we may decline your request in certain cases as permitted by law.
      </p>
      <ul>
        <li>
          <strong>Information.</strong> You can request information about how we
          have collected and used your Personal Information, including the
          categories collected, sources, purposes, and categories of third
          parties to whom we disclose it.
        </li>
        <li>
          <strong>Access.</strong> You can request a copy of the Personal
          Information that we have collected about you.
        </li>
        <li>
          <strong>Appeal.</strong> You can appeal our denial of any request
          validly submitted.
        </li>
        <li>
          <strong>Correction.</strong> You can ask us to correct inaccurate
          Personal Information that we have collected about you.
        </li>
        <li>
          <strong>Deletion.</strong> You can ask us to delete the Personal
          Information that we have collected from you.
        </li>
        <li>
          <strong>Opt-out of targeted advertising.</strong> We do not process
          your personal information for targeted advertising purposes.
        </li>
        <li>
          <strong>Opt-out of profiling / automated decision-making.</strong> We
          use automated scoring on certain writing exercises to generate a score
          and short feedback for you and your instructors. This scoring does not
          produce legal or similarly significant effects outside your course
          (such as housing, employment, or criminal justice decisions). We do
          not use profiling for those purposes.
        </li>
        <li>
          <strong>Opt-out of sales.</strong> We do not sell your Personal
          Information within the meaning of State Privacy Laws, and we do not
          share it for cross-context behavioral advertising.
        </li>
        <li>
          <strong>Consumers under 16.</strong> We do not have actual knowledge
          that we collect, sell, or share the personal information of consumers
          under 16 years of age.
        </li>
        <li>
          <strong>Sensitive Personal Information.</strong> We may process
          education records (learning activity and written work) as described in
          this Privacy Policy. We do not process Sensitive Personal Information
          for the purpose of inferring characteristics about consumers under the
          CCPA, and we do not use it for advertising.
        </li>
        <li>
          <strong>Nondiscrimination.</strong> You are entitled to exercise the
          rights described above free from discrimination as prohibited by the
          State Privacy Laws.
        </li>
      </ul>

      <h3>Exercising your rights</h3>
      <p>
        Because we do not sell or share Personal Information for advertising,
        there is no separate &quot;Do Not Sell or Share My Personal
        Information&quot; opt-out. If you broadcast a Global Privacy Control
        (GPC) signal, our default practice is already consistent with an opt-out
        of sale or sharing for targeted advertising. For more information about
        GPC, visit{" "}
        <a href="https://globalprivacycontrol.org">globalprivacycontrol.org</a>.
      </p>
      <p>
        You may submit requests to exercise any of the other state privacy
        rights listed above by emailing{" "}
        <a href={`mailto:${LEGAL_CONTACT_EMAIL}`}>{LEGAL_CONTACT_EMAIL}</a>. We
        may need to verify your identity to process access, appeal, correction,
        or deletion requests, and we reserve the right to confirm your
        residency. Under some State Privacy Laws, you may enable an authorized
        agent to make a request on your behalf. We may need to verify your
        agent&apos;s identity and authority to act on your behalf.
      </p>

      <h3>Information practices</h3>
      <p>
        The following describes our practices currently and during the past 12
        months. We collect all categories of personal information from the
        sources and use them for the business purposes described above. The
        criteria for deciding how long to retain personal information is
        generally based on whether such period is sufficient to fulfill the
        purposes for which we collected it as described in this notice,
        including complying with our legal obligations.
      </p>
      <div className="overflow-x-auto mb-4">
        <table>
          <thead>
            <tr>
              <th>Personal information we collect</th>
              <th>CCPA statutory category</th>
              <th>Purposes</th>
              <th>Disclosed for a business purpose to</th>
              <th>Sold or shared?</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>Contact data (name, email)</td>
              <td>Identifiers</td>
              <td>Service delivery, enrollment, communications</td>
              <td>
                Authentication provider, hosting provider, course
                instructors/administrators
              </td>
              <td>No</td>
            </tr>
            <tr>
              <td>Account and profile data</td>
              <td>Identifiers; customer records</td>
              <td>Authentication and account administration</td>
              <td>Authentication provider, hosting provider</td>
              <td>No</td>
            </tr>
            <tr>
              <td>Course enrollment and role</td>
              <td>Identifiers; education information</td>
              <td>Provide course access</td>
              <td>Hosting provider, course instructors/administrators</td>
              <td>No</td>
            </tr>
            <tr>
              <td>
                Learning activity (answers, scores, time on task, AI grading
                records)
              </td>
              <td>
                Education information; inferences (scores); customer records
              </td>
              <td>
                Provide and improve the Service; instructor review; AI scoring
                of writing
              </td>
              <td>
                Hosting provider, AI scoring provider, course
                instructors/administrators
              </td>
              <td>No</td>
            </tr>
            <tr>
              <td>Communications with us</td>
              <td>Identifiers; customer records</td>
              <td>Support and compliance</td>
              <td>Hosting provider (if stored)</td>
              <td>No</td>
            </tr>
            <tr>
              <td>Device, log, and cookie data</td>
              <td>Identifiers; internet or electronic activity</td>
              <td>Security, operations, keeping you signed in</td>
              <td>Authentication provider, hosting provider, font provider</td>
              <td>No</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p>
        Information you voluntarily provide to us, such as in free-form emails
        or written exercises, may contain other categories of personal
        information not described above.
      </p>

      <h3>Additional information for California residents</h3>
      <p>
        Under California&apos;s Shine the Light law (California Civil Code
        Section 1798.83), California residents may ask companies with whom they
        have formed a business relationship primarily for personal, family, or
        household purposes to provide the names of third parties to which they
        have disclosed certain personal information during the preceding
        calendar year for those third parties&apos; own direct marketing
        purposes, and the categories of personal information disclosed. We do
        not disclose personal information to third parties for their own direct
        marketing purposes. You may still send requests for this information to{" "}
        <a href={`mailto:${LEGAL_CONTACT_EMAIL}`}>{LEGAL_CONTACT_EMAIL}</a> with
        the statement &quot;Shine the Light Request,&quot; your first and last
        name and mailing address, and a certification that you are a California
        resident.
      </p>

      <h3>Additional information for Nevada residents</h3>
      <p>
        Nevada residents have the right to opt out of the sale of certain
        personal information for monetary consideration. We do not currently
        engage in such sales. If you are a Nevada resident and would like to
        make a request to opt out of any potential future sales, please email{" "}
        <a href={`mailto:${LEGAL_CONTACT_EMAIL}`}>{LEGAL_CONTACT_EMAIL}</a>.
      </p>

      <p>
        If you have questions or concerns about our privacy policies or
        information practices, please contact us using the details in the{" "}
        <a href="#contact">How to contact us</a> section above.
      </p>
    </LegalPage>
  );
}
