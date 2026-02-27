import ControlRoomDetailClient from './ControlRoomDetailClient';

interface ControlRoomDetailPageProps {
  params: Promise<{ id: string }>;
}

// Generate static params for common control rooms
export function generateStaticParams() {
  return [
    { id: 'room-main' },
    { id: 'room-cost' },
    { id: 'room-openai' },
  ];
}

export default async function ControlRoomDetailPage({ params }: ControlRoomDetailPageProps) {
  const { id } = await params;
  return <ControlRoomDetailClient id={id} />;
}
