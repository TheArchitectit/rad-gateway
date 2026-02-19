'use client';

import { useRouter } from 'next/navigation';
import { AuthLayout } from '@/components/templates/AuthLayout';
import { LoginForm } from '@/components/auth/LoginForm';

export default function LoginPage() {
  const router = useRouter();

  const handleSuccess = () => {
    router.push('/');
  };

  return (
    <AuthLayout 
      title="Sign in to RAD Gateway" 
      subtitle="Enter your credentials to access the admin console"
    >
      <LoginForm onSuccess={handleSuccess} />
    </AuthLayout>
  );
}
