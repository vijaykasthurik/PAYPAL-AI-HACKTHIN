import React, { useState } from 'react';
import {
  LandingScreen,
  AuthScreen,
  CasesDashboardScreen,
  NewPaymentScreen,
  PaymentDetailScreen,
  TimelineScreen,
  TruthGapScreen,
  SenderLensScreen,
  ReceiverLensScreen,
  SafeActionScreen,
  MirrorLabScreen,
  EvidenceScreen,
  NotificationsScreen,
  OperationsScreen,
  SettingsScreen,
  DemoWalkthroughScreen,
} from './features/screens/Screens';

export const App: React.FC = () => {
  const [currentScreen, setCurrentScreen] = useState<number>(1);
  const [currentRole, setCurrentRole] = useState<string>('sender');

  const screens = [
    { id: 1, name: '1. Landing' },
    { id: 2, name: '2. Login / Auth' },
    { id: 3, name: '3. Cases Matrix' },
    { id: 4, name: '4. New Payment / Timeout' },
    { id: 5, name: '5. Detail & Passport' },
    { id: 6, name: '6. Lifecycle Timeline' },
    { id: 7, name: '7. Truth Gap Graph' },
    { id: 8, name: '8. Sender Lens' },
    { id: 9, name: '9. Receiver Lens' },
    { id: 10, name: '10. Safe Action Engine' },
    { id: 11, name: '11. Recovery Mirror Lab' },
    { id: 12, name: '12. Evidence Explorer' },
    { id: 13, name: '13. Notifications' },
    { id: 14, name: '14. Ops & Webhooks' },
    { id: 15, name: '15. Privacy & Consent' },
    { id: 16, name: '16. Golden Demo B' },
  ];

  const renderScreen = () => {
    switch (currentScreen) {
      case 1:
        return <LandingScreen onNavigate={(s) => setCurrentScreen(s)} />;
      case 2:
        return (
          <AuthScreen
            onLogin={(r) => {
              setCurrentRole(r);
              setCurrentScreen(3);
            }}
          />
        );
      case 3:
        return <CasesDashboardScreen onSelectCase={() => setCurrentScreen(5)} />;
      case 4:
        return <NewPaymentScreen onSimulateTimeout={() => setCurrentScreen(5)} />;
      case 5:
        return <PaymentDetailScreen />;
      case 6:
        return <TimelineScreen />;
      case 7:
        return <TruthGapScreen />;
      case 8:
        return <SenderLensScreen />;
      case 9:
        return <ReceiverLensScreen />;
      case 10:
        return <SafeActionScreen />;
      case 11:
        return <MirrorLabScreen />;
      case 12:
        return <EvidenceScreen />;
      case 13:
        return <NotificationsScreen />;
      case 14:
        return <OperationsScreen />;
      case 15:
        return <SettingsScreen />;
      case 16:
        return <DemoWalkthroughScreen onRunGoldenDemo={() => setCurrentScreen(5)} />;
      default:
        return <LandingScreen onNavigate={(s) => setCurrentScreen(s)} />;
    }
  };

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      {/* Top Header */}
      <nav
        style={{
          background: 'var(--surface-2)',
          borderBottom: '1px solid var(--border)',
          padding: '12px 24px',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          position: 'sticky',
          top: 0,
          zIndex: 100,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <div
            onClick={() => setCurrentScreen(1)}
            style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' }}
          >
            <span style={{ fontSize: '22px' }}>🌉</span>
            <span style={{ fontSize: '18px', fontWeight: 800, color: 'var(--text-1)' }}>PayBridge AI</span>
          </div>
          <span className="badge badge-simulated" style={{ fontSize: '10px' }}>
            REAL SANDBOX
          </span>
        </div>

        {/* Quick Nav Dropdown */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <select
            value={currentScreen}
            onChange={(e) => setCurrentScreen(Number(e.target.value))}
            style={{
              padding: '6px 12px',
              borderRadius: '6px',
              border: '1px solid var(--border-strong)',
              fontSize: '13px',
              background: 'var(--surface-1)',
              color: 'var(--text-1)',
            }}
          >
            {screens.map((s) => (
              <option key={s.id} value={s.id}>
                Screen {s.name}
              </option>
            ))}
          </select>

          <span className="badge badge-verified" style={{ textTransform: 'capitalize' }}>
            Role: {currentRole.replace('_', ' ')}
          </span>

          <button
            onClick={() => setCurrentScreen(16)}
            style={{
              padding: '6px 14px',
              background: 'var(--accent)',
              color: '#fff',
              border: 'none',
              borderRadius: '6px',
              fontWeight: 600,
              fontSize: '12px',
            }}
          >
            Run Demo B
          </button>
        </div>
      </nav>

      {/* Main Content Area */}
      <main style={{ flex: 1, paddingBottom: '40px' }}>
        {renderScreen()}
      </main>

      {/* Footer */}
      <footer
        style={{
          textAlign: 'center',
          padding: '16px',
          borderTop: '1px solid var(--border)',
          fontSize: '12px',
          color: 'var(--text-3)',
          background: 'var(--surface-1)',
        }}
      >
        PayBridge AI — Independent Cross-Border Payment Intelligence & Safe Recovery Platform. Not affiliated with or endorsed by PayPal.
      </footer>
    </div>
  );
};
