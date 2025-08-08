<?php

namespace Omed\CMS\Entity;

use ApiPlatform\Metadata\ApiResource;
use Doctrine\ORM\Mapping as ORM;
use Gedmo\Mapping\Annotation as Gedmo;
use Gedmo\Timestampable\Traits\Timestampable;

#[ORM\Entity()]
#[ORM\Table(name: 'cms_article')]
#[ApiResource]
class Article
{
    use Timestampable;

    #[ORM\Id]
    #[ORM\GeneratedValue(strategy: 'AUTO')]
    #[ORM\Column(type: 'integer')]
    public int $id;

    #[ORM\Column(type: 'string', length: 255)]
    public string $title;

    #[ORM\Column(type: 'text', nullable: true)]
    public ?string $content = null;

    #[ORM\Column(type: 'string', length: 255, unique: true)]
    #[Gedmo\Slug(fields: ['title'])]
    public string $slug;

    #[ORM\Column(type: 'boolean')]
    public bool $published = false;

    #[ORM\Column(type: 'datetime_immutable', nullable: true)]
    #[Gedmo\Timestampable(on: 'change', field: 'published', value: true)]
    public ?\DateTimeImmutable $publishedAt = null;
}
