<?php

/*
 * This file is part of the Omed project.
 *
 * (c) Anthonius Munthi <https://itstoni.com>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

declare(strict_types=1);

namespace Omed\Laravel\User\Services;

use Doctrine\Persistence\ObjectManager;
use Illuminate\Http\Request;
use Illuminate\Support\Collection;
use LaravelDoctrine\ORM\Pagination\PaginatorAdapter;
use Omed\Component\User\Manager\UserManager as BaseUserManager;
use Omed\Component\User\Util\CanonicalFieldsUpdater;
use Omed\Component\User\Util\Canonicalizer;
use Omed\Laravel\User\Model\User;

class UserManager extends BaseUserManager
{
    public function __construct(ObjectManager $om)
    {
        /** @var string $userClass */
        /** @var \Doctrine\Persistence\ManagerRegistry $registry */
        $canonicalizer = new Canonicalizer();
        $fieldsUpdater = new CanonicalFieldsUpdater($canonicalizer, $canonicalizer);
        $userClass = config('omed.user.models.user');
        $passwordUpdater = app()->get(PasswordUpdater::class);

        parent::__construct(
            $passwordUpdater,
            $fieldsUpdater,
            $om,
            $userClass
        );
    }

    /**
     * @return Collection
     */
    public function findAll()
    {
        $data = $this->getRepository()->findAll();

        return new Collection($data);
    }

    /**
     * @return User[]
     */
    public function getUserList()
    {
        $builder = $this->getManager()->createQueryBuilder();

        $builder->select('u')
            ->from(User::class,'u');

        return PaginatorAdapter::fromRequest($builder->getQuery(),10)->make()->toArray();
    }
}
